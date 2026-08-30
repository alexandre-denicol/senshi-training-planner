ALTER TABLE training_history_participants
ADD COLUMN student_id uuid NULL REFERENCES students(id) ON DELETE SET NULL;
