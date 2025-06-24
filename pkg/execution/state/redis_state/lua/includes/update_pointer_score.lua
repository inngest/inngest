-- This function updates a function's place in the pointer queue to the given
-- score.  This score should almost always be the value from `get_fn_partition_score`.
-- It's a separate function as > 1 pointer queue may be updated at a time.
local function update_pointer_score_to(fnID, pointerQueueKey, updateTo)
    -- Only update if set.
    if updateTo > 0 then
        redis.call("ZADD", pointerQueueKey, updateTo, fnID)
    end
end

-- get_converted_earliest_pointer_score returns a high-precision queue's earliest job as a score for pointer queues.
-- Note: This operation converts high-precision item scores to lower-precision pointer scores. DO NOT USE FOR FUNCTION QUEUES.
-- This returns 0 if there are no scores available.
local function get_converted_earliest_pointer_score(keyQueueSet)
    local earliestScore = redis.call("ZRANGE", keyQueueSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
    if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
        return 0
    end
    -- queues are ordered by ms precision, whereas pointers are second precision.
    -- earliest is a table containing {item, score}
    return math.floor(tonumber(earliestScore[2]) / 1000)
end


-- get_earliest_pointer_score returns a pointer queue's earlies score. This is usually a timestamp in second precision.
-- Note: NEVER use this for high-precision scores found in function queues. This may only be used for other pointer queues.
-- This returns 0 if there are no scores available.
local function get_earliest_pointer_score(keyPointerQueueSet)
    local earliestScore = redis.call("ZRANGE", keyPointerQueueSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
    if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
        return 0
    end
    -- queues are ordered by ms precision, whereas pointers are second precision.
    -- earliest is a table containing {item, score}
    return tonumber(earliestScore[2])
end

-- get_earliest_score returns the earliest score in a given set.
local function get_earliest_score(keyQueueSet)
    local earliestScore = redis.call("ZRANGE", keyQueueSet, "-inf", "+inf", "BYSCORE", "LIMIT", 0, 1, "WITHSCORES")
    if earliestScore == nil or earliestScore == false or earliestScore[2] == nil then
        return 0
    end
    -- earliest is a table containing {item, score}
    return tonumber(earliestScore[2])
end
