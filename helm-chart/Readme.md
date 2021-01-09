# Helm Chart for Presto Operator

This is the Helm chart for the Presto operator.

## Installing the Chart

The instructions to install the Presto operator chart:

Install presto operator using local repo with command:

    ```bash
    helm install [RELEASE_NAME] . --set operatorImage.name=[IMAGE_NAME]
    ```
## Uninstalling the Chart

To uninstall your release:

  ```bash
  helm delete [RELEASE_NAME]
  ```
