# Gardener qualys cloud agent (qca) extension

This extension installs the qualys cloud agent (qca) package on all nodes of a cluster.

## Runtime Notes

This extension installs a DaemonSet that starts the installation of the agent
on every node in the cluster. The pod runs privileged and installs the agent
direct on the node, so the node operating system is patched.

The OS **must** be debian based because the agent is only bundled for debian
based systems.

As the OS is patched by the daemonset you cannot uninstall it by disabling this
extension. This is a one-way operation, the agent will be installed but cannot
be uninstalled. If you really want a clean node you have to

  - disable this extension
  - replace all nodes of the cluster with new ones

The whole installation of the agent is done by the pod, when the container is
started. The DaemonSet only starts it privileged and host-mounts some paths, so
the install script can copy the agent to the host.