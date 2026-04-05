-- scripts/seed.sql
-- Populates the DB with demo rooms, schedules, and bookings for manual testing.
-- Run via: make seed

BEGIN;

-- Demo rooms
INSERT INTO rooms (id, name, description, capacity) VALUES
  ('aaaaaaaa-0001-0001-0001-aaaaaaaaaaaa', 'Loft',     'Open space room',        12),
  ('aaaaaaaa-0002-0002-0002-aaaaaaaaaaaa', 'Board',    'Boardroom with projector', 8),
  ('aaaaaaaa-0003-0003-0003-aaaaaaaaaaaa', 'Phone Box','Quiet single-person pod',  1)
ON CONFLICT DO NOTHING;

-- Schedules: Mon–Fri 09:00–18:00 for Loft and Board; Mon–Sun 08:00–20:00 for Phone Box
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time) VALUES
  ('bbbbbbbb-0001-0001-0001-bbbbbbbbbbbb', 'aaaaaaaa-0001-0001-0001-aaaaaaaaaaaa', ARRAY[1,2,3,4,5], '09:00', '18:00'),
  ('bbbbbbbb-0002-0002-0002-bbbbbbbbbbbb', 'aaaaaaaa-0002-0002-0002-aaaaaaaaaaaa', ARRAY[1,2,3,4,5], '09:00', '18:00'),
  ('bbbbbbbb-0003-0003-0003-bbbbbbbbbbbb', 'aaaaaaaa-0003-0003-0003-aaaaaaaaaaaa', ARRAY[1,2,3,4,5,6,7], '08:00', '20:00')
ON CONFLICT DO NOTHING;

COMMIT;

-- NOTE: slots are generated automatically by the background goroutine on startup.
-- If you need them immediately after seeding, restart the app container:
--   docker compose restart app
