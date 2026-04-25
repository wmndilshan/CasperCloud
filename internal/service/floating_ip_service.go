package service

import (
	"context"
	"fmt"
	"log"

	"caspercloud/internal/network"
	"caspercloud/internal/queue"
	"caspercloud/internal/repository"

	"github.com/google/uuid"
)

const (
	TaskTypeFloatingIPAssociate    = "floating_ip.associate"
	TaskTypeFloatingIPDisassociate = "floating_ip.disassociate"
)

// FloatingIPService coordinates floating IP allocation and worker-side NAT.
type FloatingIPService struct {
	repo    *repository.Repository
	pub     queue.TaskPublisher
	ingress *network.Ingress
}

func NewFloatingIPService(repo *repository.Repository, pub queue.TaskPublisher, ingress *network.Ingress) *FloatingIPService {
	return &FloatingIPService{repo: repo, pub: pub, ingress: ingress}
}

// Allocate claims one address from the global pool for the project (synchronous).
func (s *FloatingIPService) Allocate(ctx context.Context, projectID uuid.UUID) (*repository.FloatingIP, error) {
	return s.repo.AllocateFloatingIPToProject(ctx, projectID)
}

// List returns floating IPs assigned to the project.
func (s *FloatingIPService) List(ctx context.Context, projectID uuid.UUID) ([]repository.FloatingIP, error) {
	return s.repo.ListFloatingIPsForProject(ctx, projectID)
}

// RequestAssociate binds a floating IP to a running instance and enqueues NAT setup.
func (s *FloatingIPService) RequestAssociate(ctx context.Context, projectID, floatingIPID, instanceID uuid.UUID) (*InstanceActionResult, error) {
	fip, err := s.repo.TryAssociateFloatingIP(ctx, projectID, floatingIPID, instanceID)
	if err != nil {
		return nil, err
	}
	if fip.PrivateIP == nil || *fip.PrivateIP == "" {
		return nil, fmt.Errorf("instance has no ipv4 for floating ip")
	}
	task, err := s.repo.CreateTask(ctx, TaskTypeFloatingIPAssociate, projectID, instanceID, "pending")
	if err != nil {
		return nil, err
	}
	if err := s.pub.PublishTask(ctx, queue.TaskMessage{
		TaskID:       task.ID.String(),
		Type:         task.Type,
		ProjectID:    projectID.String(),
		InstanceID:   instanceID.String(),
		FloatingIPID: floatingIPID.String(),
		PublicIP:     fip.PublicIP,
		PrivateIP:    *fip.PrivateIP,
	}); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, task.ID, "failed", &msg)
		_ = s.repo.RollbackFloatingIPAssociation(ctx, projectID, floatingIPID)
		return nil, err
	}
	return &InstanceActionResult{Task: task}, nil
}

// RequestDisassociate clears the DB binding and enqueues iptables removal.
func (s *FloatingIPService) RequestDisassociate(ctx context.Context, projectID, floatingIPID uuid.UUID) (*InstanceActionResult, error) {
	pub, priv, err := s.repo.TakeDisassociateFloatingIP(ctx, projectID, floatingIPID)
	if err != nil {
		return nil, err
	}
	task, err := s.repo.CreateTaskWithoutInstance(ctx, TaskTypeFloatingIPDisassociate, projectID, "pending")
	if err != nil {
		return nil, err
	}
	if err := s.pub.PublishTask(ctx, queue.TaskMessage{
		TaskID:       task.ID.String(),
		Type:         task.Type,
		ProjectID:    projectID.String(),
		InstanceID:   "",
		FloatingIPID: floatingIPID.String(),
		PublicIP:     pub,
		PrivateIP:    priv,
	}); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, task.ID, "failed", &msg)
		return nil, err
	}
	return &InstanceActionResult{Task: task}, nil
}

// HandleAssociateTask applies iptables for an already-active floating_ips row.
func (s *FloatingIPService) HandleAssociateTask(ctx context.Context, taskID, projectID, instanceID, floatingIPID uuid.UUID) error {
	if s.ingress == nil {
		return fmt.Errorf("floating ip associate: ingress not configured on worker")
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "running", nil); err != nil {
		return err
	}
	fip, err := s.repo.GetFloatingIPForProject(ctx, projectID, floatingIPID)
	if err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if fip.Status != "active" || fip.InstanceID == nil || *fip.InstanceID != instanceID {
		msg := "floating ip not in expected active state"
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return fmt.Errorf("%s", msg)
	}
	if fip.PrivateIP == nil || *fip.PrivateIP == "" {
		msg := "floating ip missing private_ip"
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return fmt.Errorf("%s", msg)
	}
	if err := s.ingress.AssociateIP(fip.PublicIP, *fip.PrivateIP); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "succeeded", nil); err != nil {
		return err
	}
	log.Printf("caspercloud worker: floating_ip associate done fip_id=%s", floatingIPID)
	return nil
}

// HandleDisassociateTask removes iptables using addresses from the queue message (DB row already released).
func (s *FloatingIPService) HandleDisassociateTask(ctx context.Context, taskID, projectID uuid.UUID, publicIP, privateIP string) error {
	if s.ingress == nil {
		return fmt.Errorf("floating ip disassociate: ingress not configured on worker")
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "running", nil); err != nil {
		return err
	}
	if err := s.ingress.DisassociateIP(publicIP, privateIP); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateTaskStatus(ctx, projectID, taskID, "failed", &msg)
		return err
	}
	if err := s.repo.UpdateTaskStatus(ctx, projectID, taskID, "succeeded", nil); err != nil {
		return err
	}
	log.Printf("caspercloud worker: floating_ip disassociate done public=%s private=%s", publicIP, privateIP)
	return nil
}

// ReconcileActiveFloatingIngress ensures Postgres "active" rows have matching iptables rules.
func (s *FloatingIPService) ReconcileActiveFloatingIngress(ctx context.Context) error {
	if s.ingress == nil {
		return nil
	}
	rows, err := s.repo.ListFloatingIPsActiveForReconcile(ctx)
	if err != nil {
		return err
	}
	for _, b := range rows {
		ok, err := s.ingress.FloatingNATRulesPresent(b.PublicIP, b.PrivateIP)
		if err != nil {
			log.Printf("caspercloud worker: floating ip reconcile check: %v", err)
			continue
		}
		if ok {
			continue
		}
		if err := s.ingress.AssociateIP(b.PublicIP, b.PrivateIP); err != nil {
			log.Printf("caspercloud worker: floating ip reconcile apply public=%s private=%s: %v", b.PublicIP, b.PrivateIP, err)
		}
	}
	return nil
}
