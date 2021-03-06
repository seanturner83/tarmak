.. _tarmak_clusters_apply:

tarmak clusters apply
---------------------

Create or update the currently configured cluster

Synopsis
~~~~~~~~


Create or update the currently configured cluster

::

  tarmak clusters apply [flags]

Options
~~~~~~~

::

      --auto-approve                 auto approve to responses when applying cluster (default true)
      --auto-approve-deleting-data   auto approve deletion of any data as a cause from applying cluster
  -C, --configuration-only           apply changes to configuration only, by running only puppet
      --dry-run                      don't actually change anything, just show changes that would occur
  -h, --help                         help for apply
  -I, --infrastructure-only          apply changes to infrastructure only, by running only terraform
  -P, --plan-file-location string    location of stored terraform plan executable file to be used (default "${TARMAK_CONFIG}/${CURRENT_CLUSTER}/terraform/tarmak.plan")
  -W, --wait-for-convergence         wait for wing convergence on applied instances (default true)

Options inherited from parent commands
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

  -c, --config-directory string                          config directory for tarmak's configuration (default "~/.tarmak")
      --current-cluster string                           override the current cluster set in the config
      --ignore-missing-public-key-tags ssh_known_hosts   ignore missing public key tags on instances, by falling back to populating ssh_known_hosts with the first connection (default true)
      --keep-containers                                  do not clean-up terraform/packer containers after running them
      --public-api-endpoint                              Override kubeconfig to point to cluster's public API endpoint
  -v, --verbose                                          enable verbose logging
      --wing-dev-mode                                    use a bundled wing version rather than a tagged release from GitHub

SEE ALSO
~~~~~~~~

* `tarmak clusters <tarmak_clusters.html>`_ 	 - Operations on clusters

