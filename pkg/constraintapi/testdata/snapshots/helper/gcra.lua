local function rateLimit(key, now_ns, period_ns, limit, burst, quantity)
	local result = {}
	result["limit"] = burst + 1
	local emission = period_ns / math.max(limit, 1)
	result["ei"] = emission
	result["retry_at"] = now_ns + emission
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt
	local tat = call("GET", key)
	if not tat then
		tat = now_ns
	else
		tat = tonumber(tat)
	end
	result["tat"] = tat
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end
	local increment = quantity * emission
	result["inc"] = increment
	local new_tat = tat + increment
	if now_ns > tat then
		new_tat = now_ns + increment
	end
	result["ntat"] = new_tat
	local ttl = tat - now_ns
	result["reset_after"] = ttl
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens
	local allow_at = new_tat - dvt
	result["aat"] = allow_at
	local diff = now_ns - allow_at
	result["diff"] = diff
	if diff < 0 then
		if increment <= dvt then
			result["retry_after"] = -diff
			result["retry_at"] = now_ns - diff
		end
		if origQuantity > 0 then
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true
			return result
		end
	end
	if origQuantity > 0 then
		ttl = new_tat - now_ns
		result["reset_after"] = ttl
		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens
		local expiry = string.format("%d", math.max(ttl / 1000000000, 1))
		call("SET", key, new_tat, "EX", expiry)
	end
	local next = dvt - ttl
	result["next"] = next
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	return result
end
local function throttle(key, now_ms, period_ms, limit, burst, quantity)
	local result = {}
	result["limit"] = burst + 1
	local emission = period_ms / math.max(limit, 1)
	result["ei"] = emission
	result["retry_at"] = now_ms + emission
	local dvt = emission * (burst + 1)
	result["dvt"] = dvt
	local tat = call("GET", key)
	if not tat then
		tat = now_ms
	else
		tat = tonumber(tat)
	end
	result["tat"] = tat
	local origQuantity = quantity
	if quantity == 0 then
		quantity = 1
	end
	local increment = quantity * emission
	result["inc"] = increment
	local new_tat = tat + increment
	if now_ms > tat then
		new_tat = now_ms + increment
	end
	result["ntat"] = new_tat
	local ttl = tat - now_ms
	result["reset_after"] = ttl
	local used_tokens = math.min(math.ceil(ttl / emission), limit)
	result["u"] = used_tokens
	local allow_at = new_tat - dvt
	result["aat"] = allow_at
	local diff = now_ms - allow_at
	result["diff"] = diff
	if diff < 0 then
		if increment <= dvt then
			result["retry_after"] = -diff
			result["retry_at"] = now_ms - diff
		end
		if origQuantity > 0 then
			local next = dvt - ttl
			result["next"] = next
			result["remaining"] = 0
			result["limited"] = true
			return result
		end
	end
	if origQuantity > 0 then
		ttl = new_tat - now_ms
		result["reset_after"] = ttl
		used_tokens = math.min(math.ceil(ttl / emission), limit)
		result["u"] = used_tokens
		local expiry = string.format("%d", math.max(ttl / 1000, 1))
		call("SET", key, new_tat, "EX", expiry)
	end
	local next = dvt - ttl
	result["next"] = next
	if next > -emission then
		local remaining = math.floor(next / emission)
		result["remaining"] = remaining
	end
	return result
end