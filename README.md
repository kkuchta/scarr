# Scarr

- *S* 3
- *C* loudflair
- *A* cm
- *R* oute53
- *R* edundant letter to prevent name collisions

This is completely in development right now, so the README will be more notes to myself than anything else for the immediate future.

    go run scarr.go

#  Ideal workflow:

    $ tool init kevin_blog --domain=kevinkuchta.com
    $ cd kevin_blog
    $ echo 'hello world' > index.html
    $ tool deploy
      Checking domain...done
      'kevinkuchta.com' is available. Register via Route53 for $10/yr? [y/N] y
      Registering 'kevinkuchta.com'...done
      Creating S3 bucket kevinkuchta_com...done
      Syncing to S3...done
      Creating Cloudfront Distribution...done
      Creating ACM SSL certificate...done
      
      kevin_blog now available at https://kevinkuchta.com
    $ echo 'greetings planet' > index.html
    $ tool deploy
      Checking domain...done
      Syncing to S3...done
      1 file changed
      Creating cloudfront invalidation for 1 file...done
