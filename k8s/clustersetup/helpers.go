// Package k8shard provides automation for setting up Kubernetes clusters.
// helpers.go contains helper functions for generating configuration and service files.
package clustersetup

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// generateEncryptionConfig creates the encryption configuration file.
func (cm *ClusterManager) generateEncryptionConfig(workDir string) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}
	encodedKey := base64.StdEncoding.EncodeToString(key)

	config := fmt.Sprintf(`kind: EncryptionConfig
apiVersion: v1
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: %s
      - identity: {}
`, encodedKey)

	return cm.writeFile(filepath.Join(workDir, "encryption-config.yaml"), config)
}

// generateKubeconfig creates a kubeconfig file for the specified user or component.
func (cm *ClusterManager) generateKubeconfig(workDir, name, ip string) error {
	clusterIP := cm.config.Controller.IPAddress
	if ip != "" {
		clusterIP = ip
	}

	config := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority: %s/ca.pem
    server: https://%s:6443
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
    client-certificate: %s/%s.pem
    client-key: %s/%s-key.pem
`, workDir, clusterIP, cm.config.ClusterName, cm.config.ClusterName, name, name, name, name, workDir, name, workDir, name)

	return cm.writeFile(filepath.Join(workDir, name+".kubeconfig"), config)
}

// generateEtcdService generates the etcd systemd service file.
func (cm *ClusterManager) generateEtcdService(controller Node) string {
	return fmt.Sprintf(`[Unit]
Description=etcd
Documentation=https://github.com/etcd-io/etcd
After=network.target

[Service]
User=etcd
Group=etcd
Type=notify
ExecStart=/usr/local/bin/etcd \
  --name %s \
  --cert-file=/etc/etcd/kubernetes.pem \
  --key-file=/etc/etcd/kubernetes-key.pem \
  --peer-cert-file=/etc/etcd/kubernetes.pem \
  --peer-key-file=/etc/etcd/kubernetes-key.pem \
  --trusted-ca-file=/etc/etcd/ca.pem \
  --peer-trusted-ca-file=/etc/etcd/ca.pem \
  --client-cert-auth \
  --data-dir=/var/lib/etcd
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, controller.Name)
}

// generateContainerdConfig generates the containerd configuration.
func (cm *ClusterManager) generateContainerdConfig() string {
	return `version = 2
[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".containerd]
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            SystemdCgroup = true
`
}

// generateAPIServerService generates the kube-apiserver systemd service file.
func (cm *ClusterManager) generateAPIServerService() string {
	return fmt.Sprintf(`[Unit]
Description=Kubernetes API Server
Documentation=https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
After=network.target

[Service]
ExecStart=/usr/local/bin/kube-apiserver \
  --advertise-address=%s \
  --allow-privileged=true \
  --apiserver-count=1 \
  --authorization-mode=Node,RBAC \
  --bind-address=0.0.0.0 \
  --client-ca-file=/var/lib/kubernetes/ca.pem \
  --enable-admission-plugins=NodeRestriction \
  --etcd-cafile=/var/lib/kubernetes/ca.pem \
  --etcd-certfile=/var/lib/kubernetes/kubernetes.pem \
  --etcd-keyfile=/var/lib/kubernetes/kubernetes-key.pem \
  --etcd-servers=https://%s:2379 \
  --encryption-provider-config=/var/lib/kubernetes/encryption-config.yaml \
  --kubelet-certificate-authority=/var/lib/kubernetes/ca.pem \
  --kubelet-client-certificate=/var/lib/kubernetes/kubernetes.pem \
  --kubelet-client-key=/var/lib/kubernetes/kubernetes-key.pem \
  --service-account-key-file=/var/lib/kubernetes/service-account.pem \
  --service-cluster-ip-range=%s \
  --service-node-port-range=30000-32767 \
  --tls-cert-file=/var/lib/kubernetes/kubernetes.pem \
  --tls-private-key-file=/var/lib/kubernetes/kubernetes-key.pem
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cm.config.Controller.IPAddress, cm.config.Controller.IPAddress, cm.config.ServiceCIDR)
}

// generateControllerManagerService generates the kube-controller-manager systemd service file.
func (cm *ClusterManager) generateControllerManagerService() string {
	return fmt.Sprintf(`[Unit]
Description=Kubernetes Controller Manager
Documentation=https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
After=network.target

[Service]
ExecStart=/usr/local/bin/kube-controller-manager \
  --bind-address=0.0.0.0 \
  --cluster-cidr=%s \
  --leader-elect=true \
  --service-account-private-key-file=/var/lib/kubernetes/service-account-key.pem \
  --service-cluster-ip-range=%s \
  --use-service-account-credentials=true \
  --v=2 \
  --kubeconfig=/var/lib/kubernetes/kube-controller-manager.kubeconfig
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cm.config.PodCIDR, cm.config.ServiceCIDR)
}

// generateSchedulerService generates the kube-scheduler systemd service file.
func (cm *ClusterManager) generateSchedulerService() string {
	return `[Unit]
Description=Kubernetes Scheduler
Documentation=https://kubernetes.io/docs/reference/command-line-tools-reference/kube-scheduler/
After=network.target

[Service]
ExecStart=/usr/local/bin/kube-scheduler \
  --bind-address=0.0.0.0 \
  --leader-elect=true \
  --v=2 \
  --kubeconfig=/var/lib/kubernetes/kube-scheduler.kubeconfig
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`
}

// generateContainerdService generates the containerd systemd service file.
func (cm *ClusterManager) generateContainerdService() string {
	return `[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target

[Service]
ExecStart=/bin/containerd
Restart=on-failure
RestartSec=5
Delegate=yes
KillMode=process
OOMScoreAdjust=-999
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity

[Install]
WantedBy=multi-user.target
`
}

// generateKubeletService generates the kubelet systemd service file.
func (cm *ClusterManager) generateKubeletService(worker Node) string {
	return fmt.Sprintf(`[Unit]
Description=Kubernetes Kubelet
Documentation=https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
After=containerd.service
Requires=containerd.service

[Service]
ExecStart=/usr/local/bin/kubelet \
  --config=/var/lib/kubelet/kubelet-config.yaml \
  --container-runtime-endpoint=unix:///var/run/containerd/containerd.sock \
  --image-pull-progress-deadline=2m \
  --kubeconfig=/var/lib/kubelet/%s.kubeconfig \
  --network-plugin=cni \
  --register-node=true \
  --v=2
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, worker.Name)
}

// generateKubeProxyService generates the kube-proxy systemd service file.
func (cm *ClusterManager) generateKubeProxyService() string {
	return `[Unit]
Description=Kubernetes Kube Proxy
Documentation=https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/
After=network.target

[Service]
ExecStart=/usr/local/bin/kube-proxy \
  --config=/var/lib/kube-proxy/kube-proxy-config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`
}

// generateBridgeNetworkConfig generates the CNI bridge configuration.
func (cm *ClusterManager) generateBridgeNetworkConfig(podCIDR string) string {
	return fmt.Sprintf(`{
  "cniVersion": "0.4.0",
  "name": "bridge",
  "type": "bridge",
  "bridge": "cni0",
  "isGateway": true,
  "ipMasq": true,
  "ipam": {
    "type": "host-local",
    "ranges": [
      [{"subnet": "%s"}]
    ],
    "routes": [{"dst": "0.0.0.0/0"}]
  }
}
`, podCIDR)
}

// generateLoopbackNetworkConfig generates the CNI loopback configuration.
func (cm *ClusterManager) generateLoopbackNetworkConfig() string {
	return `{
  "cniVersion": "0.4.0",
  "name": "loopback",
  "type": "loopback"
}
`
}

// generateKubeletConfig generates the kubelet configuration.
func (cm *ClusterManager) generateKubeletConfig(worker Node) string {
	return fmt.Sprintf(`apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
address: %s
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
authorization:
  mode: Webhook
clusterDNS:
- %s
clusterDomain: cluster.local
podCIDR: %s
resolvConf: /etc/resolv.conf
`, worker.IPAddress, cm.config.ClusterDNS, worker.PodCIDR)
}

// generateKubeProxyConfig generates the kube-proxy configuration.
func (cm *ClusterManager) generateKubeProxyConfig() string {
	return fmt.Sprintf(`apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
clientConnection:
  kubeconfig: /var/lib/kube-proxy/kube-proxy.kubeconfig
mode: iptables
clusterCIDR: %s
`, cm.config.PodCIDR)
}

// generateCoreDNSManifest generates the CoreDNS manifest.
func (cm *ClusterManager) generateCoreDNSManifest() string {
	return fmt.Sprintf(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: coredns
rules:
- apiGroups: [""]
  resources: ["endpoints", "services", "pods", "namespaces"]
  verbs: ["list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: coredns
subjects:
- kind: ServiceAccount
  name: coredns
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
spec:
  replicas: 2
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      serviceAccountName: coredns
      containers:
      - name: coredns
        image: coredns/coredns:%s
        args:
        - -conf
        - /etc/coredns/Corefile
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
      volumes:
      - name: config-volume
        configMap:
          name: coredns
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health
        kubernetes cluster.local in-addr.arpa ip6.arpa {
          pods insecure
          fallthrough in-addr.arpa ip6.arpa
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
spec:
  clusterIP: %s
  ports:
  - name: dns
    port: 53
    protocol: UDP
  - name: dns-tcp
    port: 53
    protocol: TCP
  - name: metrics
    port: 9153
    protocol: TCP
  selector:
    k8s-app: kube-dns
`, cm.config.CoreDNSVersion, cm.config.ClusterDNS)
}

// generateTestApplicationManifest generates a test application manifest.
func (cm *ClusterManager) generateTestApplicationManifest() string {
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  selector:
    app: test-app
`
}

// writeFile writes content to a file in the working directory.
func (cm *ClusterManager) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}