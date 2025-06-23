I'm working on the queue, which is stored in @pkg/execution/state/redis_state/

The queue is built on Redis and works in the following way

- Queue items are added to their respective backlog by Enqueue
- A scanner runs over backlogs and checks how many items can be added into a ready queue. This is determined by checking capacity based on constraints.
  - Possible constraints are concurrency and throttles.
  - Concurrency comes on multiple levels
    - Account concurrency
    - Function concurrency
    - Custom concurrency keys: Users can add expressions, which are evaluated on each event to generate a dynamic key (e.g. event.data.userID which will create a separate limit for each user)
- The 
- The ready queue is scanned by another processor which will Lease and start processing items. Leased items are moved out of the ready queue into an in progress queue. Leasing also updates account and custom concurrency in progress ZSETs.
- After processing, items are dequeued (removed from the queue altogether) or requeued (moved back to the backlog)


I am trying to find out why there are active sets in production with hanging items.                                                                         

Items are added to active sets in @pkg/execution/state/redis_state/lua/queue/lease.lua and @pkg/execution/state/redis_state/lua/queue/backlog.                            

Items are removed from active sets in @pkg/execution/state/redis_state/lua/queue/requeue.lua and @pkg/execution/state/redis_state/lua/queue/dequeue.lua                         

The feature flag to enable key queues can be turned on and off multiple times, so while rolling out, we had to roll back a couple times to fix bugs, then re-enrolled users.

To fix this issue, I've created an active checker in @pkg/execution/state/redis_state/active_checker.go. 

This is used the following way                                              
- Every time the queue scans backlogs (collections of items that could be moved to the ready queue, from which items are picked for processing) and runs into concurrency limits, we add a reference to the backlog (its ID) to a Redis ZSET.
- This ZSET is scanned by a new entrypoint, the active scanner. It picks up multiple backlogs at a time and starts working on them
- For each backlog, we scan active sets on the account level, the function level, and custom concurrency keys.

Think hard about cases where we add items to active sets without removing them.
