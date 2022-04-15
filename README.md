# eventexporter

[![Open in Visual StudioCode](https://open.vscode.dev/badges/open-in-vscode.svg)](https://open.vscode.dev/champly/eventexporter)

Multi kubernetes cluster event exporter to sink (alertmanager).

### build

You can build within your local `make build` or build within docker `make docker-build`. If you want push with your own repo, modify `Makefile` `IMAGE_REG` or set env, then use `make docker-build-push` or `make build-push`.

### Configuration

If you run within your local env, you can define your own config like
[config.yaml](https://github.com/champly/eventexporter/blob/main/config/config.yaml) . And run it with
`--exporter_config_path` which is your config path.

If you run within kubernetes cluster, you can change Configmap with your own rule.

### Compare with [kubernetes-event-exporter](https://github.com/opsgenie/kubernetes-event-exporter)

| feature              | kubernetes-event-exporter                                                    | evenexporter |
| :--:                 | :--:                                                                         | :--:         |
| multi sink           | [multi](https://github.com/opsgenie/kubernetes-event-exporter#configuration) | alertmanager |
| enhanced event cache | ❌                                                                           | ✅           |
| multi cluster        | ❌                                                                           | ✅           |
