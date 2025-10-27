package constraintapi

/*
*	local requestData
*	local latestConfig
*
* handleIdempotency()
*		... might return if request was successfully completed within idempotency TTL
*
*	local reserved = {
*
*
*	}
*
*
* if requestData.rateLimit {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.throttle {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
* if requestData.accountConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.functionConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
* if requestData.customConcurrency {
*			local remainingCapacity = (limit(latestConfig, "accountConcurrency") - current("accountConcurrency"))
*		reserved["accountConcurrency"] = remainingCapacity
*
	* }
*
*
*
* if isEmpty(reserved) {
*		// return "no capacity left, please wait for a bit"
* }
*
* -- At this point, we know that the request reserved _some_ capacity
*
*	redis.call("ZADD", leasesSet, leaseID, leaseExpiry)
*
*	redis.call("HSET", leasesHash, leaseID, requestData)
*	redis.call("HSET", leaseReserved, leaseID, reserved )
*
*	return {
*		leaseID,
*		reserved, -- client should know how much capacity was _actually_ reserved
*
*	}
*
*
*/
