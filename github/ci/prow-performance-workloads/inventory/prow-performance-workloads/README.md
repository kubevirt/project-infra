Edited the following files and parameters from the example from the kubespray master repository.

Edit the dedicated resources to master nodes and kubelet
system_reserved: true
1GMem and 1000m=1CPUs to kubelet
25GMem and 25000m=25CPUs to masters

Edited the container runtime, docker has low performance
container_manager: containerd

Configure calico_iptables_backend: "Auto" in group_vars/k8s_cluster/k8s-net-calico.yml

Enable Internal Registry
registry_enabled: true and local_volume_provisioner_enabled: true in group_vars/k8s_cluster/addons.yml 

Change the number of pods per node
group_vars/k8s_cluster/k8s-cluster.yml kubelet_max_pods
kube_network_node_prefix: 23
kubelet_max_pods: 510

Increased kubeAPIQPS and kubeAPIBurst values to:
group_vars/kube-proxy.yml
kube_proxy_client_qps: 50
kube_proxy_client_burst: 100