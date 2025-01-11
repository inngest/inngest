--[[

Saves a response for a step.  This automatically creates history entries
depending on the response being saved.

Output:
 -1: duplicate response
  0: Successfully saved response

]]

local keyMetadata = KEYS[1]
local keyKV 	  = KEYS[2]

local key   = ARGV[1]
local value = ARGV[2]

redis.call("HSET", keyMetadata, "usesKV", "1")
redis.call("HSET", keyKV, key, value)

return 0
