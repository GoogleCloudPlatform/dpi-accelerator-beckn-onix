controller:
  replicaCount: 3
  admissionWebhooks:  # This section is optional
    enabled: false
  service:
    type: ClusterIP
    annotations:
      cloud.google.com/neg: '{"exposed_ports": {"80": {"name": "${neg_name}"}}}'
