WITH X AS (SELECT Run_ID AS Foo, App_ID FROM RUNS) SELECT x.FOO, app_id, (SELECT count(*) FROM "Events" e WHERE e.ID = X.Foo) AS n FROM x
