package kubelet

//internal/
//  node/          # node registration + heartbeat
//  podmanager/    # local cache of assigned pods (desired state)
//  syncloop/      # reconcile desired vs actual, per-pod workers
//  runtime/       # container runtime abstraction
//  status/        # pod status reporting back to API server
//  apiclient/     # watch + patch wrapper around API server
//  config/        # flags, kubeconfig loading
