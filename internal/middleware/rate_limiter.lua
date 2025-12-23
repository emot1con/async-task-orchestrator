-- KEYS[1] = key rate limit (misal: rate:user:123)
-- ARGV[1] = capacity
-- ARGV[2] = refill_rate (token per detik)
-- ARGV[3] = now (epoch seconds)

local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local data = redis.call("HMGET", KEYS[1], "tokens", "last_refill")
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

-- inisialisasi
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- refill token
local delta = now - last_refill
if delta > 0 then
    local refill = delta * refill_rate
    tokens = math.min(capacity, tokens + refill)
    last_refill = now
end

-- cek token
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call("HMSET", KEYS[1],
    "tokens", tokens,
    "last_refill", last_refill
)

-- optional TTL biar Redis bersih
redis.call("EXPIRE", KEYS[1], math.ceil(capacity / refill_rate))

return allowed
