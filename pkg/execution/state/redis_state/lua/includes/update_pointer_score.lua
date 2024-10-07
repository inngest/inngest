-- This function updates a function's place in the pointer queue to the given
-- score.  This score should almost always be the value from `get_fn_partition_score`.
-- It's a separate function as > 1 pointer queue may be updated at a time.
local function update_pointer_score_to(fnID, pointerQueueKey, updateTo)
    -- Only update if set.
    if updateTo > 0 then
        redis.call("ZADD", pointerQueueKey, updateTo, fnID)
    end
end

-- get_fn_partition_score returns a fn's earliest job as a score for pointer queues.
-- This returns 0 if there are no scores available.
local function get_fn_partition_score(fnQueueKey)
    local earliestScore = redis.call("ZRANGE", fnQueueKey, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
    if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
        return 0
    end
    -- queues are ordered by ms precision, whereas pointers are second precision.
    -- earliest is a table containing {item, score}
    return math.floor(tonumber(earliestScore[2]) / 1000)
end
