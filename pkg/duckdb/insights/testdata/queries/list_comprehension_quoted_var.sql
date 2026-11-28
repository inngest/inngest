SELECT [1 FOR "x IN [1]] a, (SELECT count(*) FROM inngest.runs) n, [1 FOR y" IN [1]] c FROM runs
