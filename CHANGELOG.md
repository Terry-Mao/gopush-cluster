## Version 1.1 (pending)

New Features:

 - publish message support.
 - group message support.
 - Migrate support.

Bugfixes:

 - 

## Version 1.0.5 (2014-09-29)

Changes:
 - add new ketama consistency hash with weight setting (modify comet, message).
 - add mutiple-push API: /1/admin/push/mprivate.
 - using weight random arithmetic instead the message node seleted
 - update document.

Bugfixes:

 - Fixed mysql fetch data bugs.
 - Fixed old version protocol compatibility.
 - Fixed rpc connection flash off when node changed

## Version 1.0.4 (2014-05-28)

Changes:

Bugfixes:

 - Fixed /1/time/get http handler timeid value error
 - Fixed HandleWrite goroutine memory leak.
 - Fixed old version protocol compatibility.

## Version 1.0.3 (2014-04-30)

Changes:

 - Refactor message module, add more log.
 - Refactor web module, add more log, http api add version in url.
 - Refactor comet module, add more log.

New Features:


Bugfixes:

 - Fixed redis clean expired message function connection leak bug.


## Version 1.0.2 (2014-04-29)

Changes:

New Features:

Bugfixes:

  - Fixed comet old protocol bug.

## Version 1.0 (2014-04-28)

Initial Release

