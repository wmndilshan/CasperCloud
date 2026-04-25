-- Keep task rows after the instance is deleted (e.g. instance.destroy) so status can remain succeeded.
-- IPAM slots use ON DELETE SET NULL on instance_id; deleting the instance row still releases the slot.

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_instance_id_fkey;

ALTER TABLE tasks
    ALTER COLUMN instance_id DROP NOT NULL;

ALTER TABLE tasks
    ADD CONSTRAINT tasks_instance_id_fkey
    FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE SET NULL;
