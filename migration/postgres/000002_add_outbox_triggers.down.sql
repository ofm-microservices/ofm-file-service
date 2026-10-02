DROP TRIGGER IF EXISTS files_outbox_event ON files;
DROP FUNCTION IF EXISTS emit_file_outbox_event();
DROP TABLE IF EXISTS outbox_events;
