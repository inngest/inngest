--[[
Atomically save a Defer record, refusing to resurrect an aborted one and
refusing to add new defers beyond the per-run limit.

Stores meta and Input under separate hash keys so SetDeferStatus never
round-trips Input through cjson (which corrupts nested empty objects and
loses precision on integers above 2^53), and so the SDKRequest path can
read just the meta hash without paying the input transfer cost.

Without atomicity a retried DeferAdd can undo an interleaved DeferCancel:
T1 SaveDefer → T2 SetDeferStatus(Aborted) → T3 retry of T1 would silently
overwrite. Reading and writing inside one Lua invocation closes the race.
The two keys share a `{...}` hash tag so both writes execute in the same
cluster slot inside this script.

An aborted record is sticky for the lifetime of the run: any subsequent
SaveDefer for the same hashedID is a deliberate no-op, including a hypothetical
"cancel then re-add" pattern. Re-adding after cancel within a run is not a
supported SDK flow: same hashedID + cancel is final.

The per-run limit applies only to *new* hashedIDs. Re-saves of an existing
hashedID (legitimate SDK retransmits) are always allowed through.

KEYS[1] - defers meta hash key
KEYS[2] - defers input hash key
ARGV[1] - hashedID
ARGV[2] - meta JSON ({FnSlug, HashedID, ScheduleStatus} only)
ARGV[3] - raw Input bytes (HSET verbatim, never decoded by Lua)
ARGV[4] - integer ScheduleStatusAborted
ARGV[5] - integer max defers per run

Output:
   1: written
   0: no-op (existing record is already aborted)
  -1: no-op (per-run defer limit exceeded)
]]

local metaKey      = KEYS[1]
local inputKey     = KEYS[2]
local hashedID     = ARGV[1]
local metaPayload  = ARGV[2]
local inputPayload = ARGV[3]
local abortedValue = tonumber(ARGV[4])
local maxDefers    = tonumber(ARGV[5])

local existing = redis.call("HGET", metaKey, hashedID)
if existing then
    local prev = cjson.decode(existing)
    if prev.ScheduleStatus == abortedValue then
        return 0
    end
else
    -- New defer; enforce the per-run limit. The meta hash holds exactly one
    -- field per defer, so HLEN is the current defer count.
    local total = redis.call("HLEN", metaKey)
    if total >= maxDefers then
        return -1
    end
end

redis.call("HSET", metaKey, hashedID, metaPayload)
redis.call("HSET", inputKey, hashedID, inputPayload)
return 1
