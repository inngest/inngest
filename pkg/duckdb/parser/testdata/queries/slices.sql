SELECT x[2], x[2:], x[:], x[1:3] FROM t;

SELECT x[1:-], x[1:-:2], x[1:2:3], x[::2] FROM t;
