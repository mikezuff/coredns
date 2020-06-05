# llnwdebug

## Name

*llnwdebug* - serves an HTTP endpoint with DNS debug information.

## Description

The *llnwdebug* plugin serves an HTTP endpoint that is accessed via a unique hostname.
Information about the DNS request for that hostname is returned via the endpoint.

## Syntax

~~~ txt
llnwdebug
~~~

## Examples

This plugin provides two endpoints in any zone which it is applied.

The main endpoint is `/resolverinfo`. This endpoint is used with a unique hostname in the zone.
When the unique hostname is resolved over DNS, the DNS server records information about the
resolver. That information is returned via HTTP.

The address of the resolver is returned.

The second endpiont is `/redirect`, which returns redirects to a unique hostname for the second
endpoint.

Running configuration
~~~ corefile
ri.llnwi.com {
    llnwdebug
}
~~~

then requesting http://ri.llnwi.com/redirect might redirect to
http://ri-1590965015-1d729566.ri.llnwi.com/resolverinfo which returns

~~~ json
{"resolver":{"addr":"198.36.160.3"},"client":{"addr":"97.115.237.219"}}
~~~
