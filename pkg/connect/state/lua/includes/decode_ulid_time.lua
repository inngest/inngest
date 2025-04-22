-- This table is used when decoding ulid timestamps.
local ulidMap = { ["0"] = 0, ["1"] = 1, ["2"] = 2, ["3"] = 3, ["4"] = 4, ["5"] = 5, ["6"] = 6, ["7"] = 7, ["8"] = 8, ["9"] = 9, ["A"] = 10, ["B"] = 11, ["C"] = 12, ["D"] = 13, ["E"] = 14, ["F"] = 15, ["G"] = 16, ["H"] = 17, ["J"] = 18, ["K"] = 19, ["M"] = 20, ["N"] = 21, ["P"] = 22, ["Q"] = 23, ["R"] = 24, ["S"] = 25, ["T"] = 26, ["V"] = 27, ["W"] = 28, ["X"] = 29, ["Y"] = 30, ["Z"] = 31 }

--- decode_ulid_time decodes a ULID into a ms epoch
local function decode_ulid_time(s)
	if #s < 10 then
		return 0
	end

	-- Take first 10 characters of the ULID, which is the time portion.
	s = string.sub(s, 1, 10)
	local rev = tostring(s.reverse(s))
	local time = 0
	for i = 1, #rev do
		time = time + (ulidMap[string.sub(rev, i, i)] * math.pow(32, i-1))
	end
	return time
end
