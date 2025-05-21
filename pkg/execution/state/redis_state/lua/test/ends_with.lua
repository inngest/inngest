local key = KEYS[1]

-- $include(ends_with.lua)

if exists_without_ending(key, ":-") then
  return 1
else
  return 0
end
