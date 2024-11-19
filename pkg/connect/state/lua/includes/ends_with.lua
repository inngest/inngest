local function ends_with(str, ending)
   return ending == "" or str:sub(-#ending) == ending
end

-- used to ensure that keys don't terminate in a specific string, but still exist.
local function exists_without_ending(str, ending)
   return str ~= "" and str ~= nil and ends_with(str, ending) == false
end
