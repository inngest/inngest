# Runner

The concept of a `runner` is, well, legacy.  It still exists and is the entrypoint to the
self-hosting service, HOWEVER:

* In cloud, we split up new runs, pauses, execution, and so on into separate services
* We should OSS those services, then combine them via `inngest serve`.  The serve command
  can eg. `inngest serve new-runs pauses crons` to run more than one service.
* We can then allow each service to scale independently, and consolidate the concept
  of services between the cloud and OSS

In short, we should eventually remove this package and make the OSS stuff nicer, then
use the OSS stuff with the cloud additions in the future.
