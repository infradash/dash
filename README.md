Dash
=====

## About

Dash is an agent that helps manage a fleet of server applications running in Docker containers:

  +  monitors when new containers are started and tracks them by registering the arrival of the container
instances in a centralized registry (like Zookeeper).
  + coordinates with the central configuration store (Zookeeper) to find out the latest software builds
and downloads them locally.
  + is responsible for making sure there's enough space (e.g. disk space) to host these containers.

Dash is to be run inside another container but coordinates with the docker daemon on each host.  Only one
container instance of dash is needed per host.

Dash can also run inside a container as process launcher.  When running inside a container, it can export
environment variables to the application process by sourcing these environment variable values from a
centralized configuration store like Zookeeper.

Dash also contains a set of REST endpoints that allow UI dashboard to be built.

## Roadmap

  + Add UI for
    + Editing of environment variables in Zookeeper
    + Visualization of servers and the containers running on them
    + Visualization of containers instance statuses