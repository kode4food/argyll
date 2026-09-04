-- membership tests for the Space match document, which exposes tags as an
-- array; Lua has no built-in contains
function has(list, value)
  for _, v in ipairs(list or {}) do
    if v == value then return true end
  end
  return false
end

function has_prefix(list, prefix)
  for _, v in ipairs(list or {}) do
    if type(v) == "string" and v:sub(1, #prefix) == prefix then
      return true
    end
  end
  return false
end

-- a script reads a library table through its own environment but writes
-- straight to the shared one, so each is replaced by a read-only stand-in
for _, name in ipairs({"string", "table", "math", "bit32"}) do
  local real = _ENV[name]
  if real then
    _ENV[name] = setmetatable({}, {
      __index = real,
      __newindex = function() error(name .. " is read-only") end,
      __metatable = false,
    })
  end
end
