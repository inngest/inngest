-- UpdateMetadata updates a run's metadata.

local keyMetadata = KEYS[1]

local ctx      = ARGV[1]
local debugger = ARGV[2]
local die      = ARGV[3] -- disable immediate execution
local rv       = ARGV[4] -- request version

redis.call("HSET", keyMetadata, "ctx", ctx)
redis.call("HSET", keyMetadata, "die", die)
redis.call("HSET", keyMetadata, "debugger", debugger)
redis.call("HSET", keyMetadata, "rv", rv)

return 0
