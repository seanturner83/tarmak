.. _wing_agent:

wing agent
----------

Launch Wing agent

Synopsis
~~~~~~~~


Launch Wing agent

::

  wing agent [flags]

Options
~~~~~~~

::

      --cluster-name string    this specifies the cluster name [environment]-[cluster] (default "myenv-mycluster")
  -h, --help                   help for agent
      --instance-name string   this specifies the instance's name (default "$(hostname)")
      --manifest-url string    this specifies the URL where the puppet.tar.gz can be found
      --server-url string      this specifies the URL to the wing server (default "https://localhost:9443")

SEE ALSO
~~~~~~~~

* `wing <wing.html>`_ 	 - wing is the agent component for tarmak, it runs on every instance of tarmak

