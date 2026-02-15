# mqtt2prometheus Helm Chart

This Helm chart deploys mqtt2prometheus to Kubernetes.

## Configuration

### Application Configuration

The mqtt2prometheus configuration can be specified directly in the `values.yaml` file under the `config` section. This configuration will be automatically mounted at `/config.yaml` in the pod.

Example configuration:

```yaml
config:
  mqtt:
    server: tcp://mqtt-broker:1883
    user: myuser
    password: mypassword
    topic_path: sensors/+/data
    qos: 0
  cache:
    timeout: 24h
  json_parsing:
    separator: .
  metrics:
    - prom_name: temperature
      mqtt_name: temperature
      help: Temperature reading
      type: gauge
      const_labels:
        sensor_type: dht22
```

For a complete configuration example, see `config.yaml.dist` in the repository root.

### Installation

Install the chart with:

```bash
helm install my-mqtt2prometheus ./helm \
  --set config.mqtt.server=tcp://your-mqtt-broker:1883 \
  --set config.mqtt.topic_path=your/topic/+
```

Or create a custom `values.yaml` file and install with:

```bash
helm install my-mqtt2prometheus ./helm -f my-values.yaml
```

### Upgrading

To upgrade an existing release:

```bash
helm upgrade my-mqtt2prometheus ./helm -f my-values.yaml
```

## Additional Volumes

You can mount additional volumes (e.g., for TLS certificates) using the `volumes` and `volumeMounts` fields:

```yaml
volumes:
  - name: certs
    secret:
      secretName: mqtt-certs

volumeMounts:
  - name: certs
    mountPath: /certs
    readOnly: true
```

Then reference them in your config:

```yaml
config:
  mqtt:
    server: tcp://mqtt-broker:8883
    ca_cert: /certs/ca.pem
    client_cert: /certs/client.pem
    client_key: /certs/client-key.pem
```
