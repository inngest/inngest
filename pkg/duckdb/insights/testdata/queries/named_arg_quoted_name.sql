SELECT struct_pack("a := 1, b := (SELECT count(*) FROM inngest.runs), c" := 1) AS s FROM runs
