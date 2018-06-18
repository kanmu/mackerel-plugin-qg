# mackerel-plugin-qg

This is a custom metrics plugin for mackerel.io agent, which collects the statistics information on [qg](https://github.com/achiku/qg)

## Synopsis

```shell
mackerel-plugin-qg [-pguser=<username>]
                   [-pgpassword=<password>]
                   [-pgdatabase=<databasename>]
                   [-pgsslmode=<sslmode>]
                   [-pgsslkey=<sslkey>]
                   [-pgsslcert=<sslcert>]
                   [-pgsslrootcert=<sslrootcert>]
                   [-connect_timeout=<timeout>]
                   [-queue=<queue>]
                   [-type=<jobtype>]
                   [-metric-key-prefix=<prefix>]
```

Connection parameters such as user, password, database, and SSL configuration will be taken from environment variables if `-pguser`, `-pgpassword`, `-pgdatabase` or `-pgssl*` is unspecified.

## Example of mackerel-agent.conf

```
[plugin.metrics.qg]
command = "/path/to/mackerel-plugin-qg -pguser=test -pgpassword=secret -pgdatabase=databasename -queue=main"
```

## License

MIT License.
