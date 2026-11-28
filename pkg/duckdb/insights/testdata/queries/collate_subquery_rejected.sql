SELECT status COLLATE (SELECT name FROM events) FROM runs
