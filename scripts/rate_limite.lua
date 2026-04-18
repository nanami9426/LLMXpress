-- KEYS[1] = request key
-- KEYS[2] = token key
-- KEYS[3] = request bucket key
local req_enabled = tonumber(ARGV[1]) -- 是否使用请求数限流。ARGV[1] = req_enabled (0/1)
local req_limit = tonumber(ARGV[2]) -- 请求数上限 ARGV[2] = req_limit
local req_cost = tonumber(ARGV[3]) -- 本次请求要消耗多少额度 ARGV[3] = req_cost
local tok_enabled = tonumber(ARGV[4]) -- 是否启动token级限流 ARGV[4] = tok_enabled (0/1)
local tok_limit = tonumber(ARGV[5]) -- token数上限 ARGV[5] = tok_limit
local tok_cost = tonumber(ARGV[6]) -- 本次请求要消耗多少token ARGV[6] = tok_cost
local fixed_ttl = tonumber(ARGV[7]) -- 固定窗口计数器的过期时间 ARGV[7] = fixed_ttl_seconds
local bucket_enabled = tonumber(ARGV[8]) -- 是否使用令牌桶 ARGV[8] = bucket_enabled (0/1)
local bucket_capacity = tonumber(ARGV[9]) -- 令牌桶容量 ARGV[9] = bucket_capacity
local bucket_cost = tonumber(ARGV[10]) -- 本次请求要消耗多少桶里的token ARGV[10] = bucket_cost
local bucket_window_ms = tonumber(ARGV[11]) -- 桶从0恢复到满容量需要的时间 ARGV[11] = bucket_window_ms
local now_ms = tonumber(ARGV[12]) -- 当前时间戳 ARGV[12] = now_ms
local bucket_ttl = tonumber(ARGV[13]) -- 令牌桶状态过期的时间 ARGV[13] = bucket_ttl_seconds
local bucket_after_tokens = nil
local bucket_after_ts = nil

if req_enabled == 1 then
	local req_current = tonumber(redis.call("GET", KEYS[1]) or "0")
	if req_current + req_cost > req_limit then
		return {0, "request"}
	end
end

if tok_enabled == 1 then
	local tok_current = tonumber(redis.call("GET", KEYS[2]) or "0")
	if tok_current + tok_cost > tok_limit then
		return {0, "token"}
	end
end

if bucket_enabled == 1 then
	local state = redis.call("HMGET", KEYS[3], "tokens", "ts_ms")
	local tokens = tonumber(state[1])
	local ts = tonumber(state[2])

	if tokens == nil then
		tokens = bucket_capacity
	end
	if ts == nil then
		ts = now_ms
	end

    -- 如果Redis里的时间戳比当前时间还大，就修正为当前时间，避免时钟异常导致负时间差
	if ts > now_ms then
		ts = now_ms
	end

	local elapsed = now_ms - ts
	if elapsed > 0 then
        -- 先算桶里还剩多少token，恢复速率是bucket_capacity / bucket_window_ms
		tokens = tokens + (elapsed * bucket_capacity / bucket_window_ms)
		if tokens > bucket_capacity then
			tokens = bucket_capacity
		end
	end

	if tokens < bucket_cost then
		return {0, "request"}
	end

	bucket_after_tokens = tokens - bucket_cost
	bucket_after_ts = now_ms
end

if req_enabled == 1 then
	local req_after = redis.call("INCRBY", KEYS[1], req_cost)
	if tonumber(req_after) == req_cost then
        -- 如果加完后的值刚好等于本次消耗，说明这是第一次创建这个key
        -- 只有第一次写入时设置TTL，后续只累加，不刷新窗口
		redis.call("EXPIRE", KEYS[1], fixed_ttl)
	end
end

if tok_enabled == 1 then
	local tok_after = redis.call("INCRBY", KEYS[2], tok_cost)
	if tonumber(tok_after) == tok_cost then
		redis.call("EXPIRE", KEYS[2], fixed_ttl)
	end
end

if bucket_enabled == 1 then
	redis.call("HSET", KEYS[3], "tokens", tostring(bucket_after_tokens), "ts_ms", tostring(bucket_after_ts))
	redis.call("EXPIRE", KEYS[3], bucket_ttl)
end

return {1, ""}