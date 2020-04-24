fairy 
---------------------------------

The deployment fairy provides a simple way to manage deployments 
across multiple accounts in AWS.

```shell script
$ fairy deploy -p example -d examples/basic
2020/04/24 09:58:31
2020/04/24 09:58:31 deployment fairy started
2020/04/24 09:58:32
2020/04/24 09:58:32 uploading resources ...
2020/04/24 09:58:32 uploaded examples/basic/resources/example.txt -> s3://fairy-292985526836-us-west-2/resources/example/local/latest/example.txt (284ms) - <nil>
2020/04/24 09:58:32
2020/04/24 09:58:32 deploying cloudformation templates ...
2020/04/24 09:58:32 retrieved 2 stack summaries, (76ms, prefix: local-example-) - <nil>
2020/04/24 09:58:32 skipping update: no updates required
2020/04/24 09:58:32 updated cloudformation stack, local-example--table (126ms) - <nil>
2020/04/24 09:58:32 applied 1 cloudformation changes (126ms) - <nil>
2020/04/24 09:58:32
2020/04/24 09:58:32 deployment fairy completed - 934ms
``` 
