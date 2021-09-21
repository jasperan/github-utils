#!/bin/bash

# launch cleanup process
git filter-repo --strip-blobs-bigger-than 3M

# run GC
git reflog expire --expire=now --all && git gc --prune=now --aggressive

# update the more
git push