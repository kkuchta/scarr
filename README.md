# Scarr

- **S** 3
- **C** loudfront
- **A** CM
- **R** oute53
- **R** edundant letter to prevent name collisions

If you want to set up a production-grade flat file site, a reasonable way to accomplish this would be to load your files to S3, put cloudfront in front of that for caching, use route53 for domain registration + DNS, and ACM for your TLS certificate.  This tool automates all of that from registering the domain to uploading files.

It works like this:

```
$ scarr init -domain nogood.reisen -name nogoodreisen
Initializing...done
$ cd nogoodreisen/
$ mvim scarr.yml # Edit a few fields here
$ echo "<html>hello world</html>" > index.html
$ AWS_PROFILE=scarr scarr deploy
  ... a bunch of aws stuff happens automatically ...
$ curl https://nogood.reisen
  <html>hullo wold</html>
```
The `deploy` command does the following:

1. Registers the given domain through route53 (you'll be prompted to confirm this)
2. Creates a TLS certificate through ACM
3. Uses route53 DNS to validate that certificate
4. Creates an S3 bucket
5. Creates a cloudfront distribution pointed to that S3 bucket using the ACM certificate
6. Creates an apex dns record pointing to that cloudfront
7. Syncs the current directory to that S3 bucket and invalidates the cloudfront cache.

TLDR: Cheap, painless, fast, bulletproof flatfile sites with https and an apex domain.

# Quickstart
1. Download the binary from https://scarr.io/dist/scarr
2. Set up an aws user with the permissions listed under "Configure" below
3. Run `scarr init -domain domainyouwant.com -name mycoolproject` and cd into the generated directory
4. Create an index.html page in that directory
5. Run `AWS_ACCESS_KEY_ID=your_access_key_here AWS_SECRET_KEY_ID=your_secret_key_here scarr deploy`

And once scarr finishes deploying, your site should be live at https://domainyouwant.com

# Installation

Scarr is distributed as a simple binary that you can download [here](https://scarr.io/dist/scarr).  Right now it's only been tested on Mac OS.

```
curl https://scarr.io/dist/scarr > scarr
chmod +x scarr
./scarr init ...
```

# Configure

### AWS Credentials
You'll need an AWS IAM user with sufficient permissions.  One way to do this is to go to to https://console.aws.amazon.com/iam/home and create a new policy with the following policy json:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "route53:CreateHostedZone",
                "s3:GetBucketWebsite",
                "route53:ListHostedZones",
                "cloudfront:GetInvalidation",
                "route53:ChangeResourceRecordSets",
                "s3:CreateBucket",
                "s3:ListBucket",
                "cloudfront:CreateDistribution",
                "route53domains:GetDomainDetail",
                "s3:GetBucketAcl",
                "cloudfront:CreateInvalidation",
                "route53domains:GetOperationDetail",
                "s3:PutObject",
                "s3:GetObject",
                "route53domains:CheckDomainAvailability",
                "s3:PutBucketWebsite",
                "acm:DescribeCertificate",
                "acm:RequestCertificate",
                "route53domains:RegisterDomain",
                "cloudfront:ListDistributions",
                "route53:ListResourceRecordSets",
                "s3:PutBucketAcl",
                "acm:ListCertificates",
                "s3:PutObjectAcl"
            ],
            "Resource": "*"
        }
    ]
}
```
Then create a new IAM user and attach that policy to it.  Use that user's access key id and secret access key values to run scarr:

`AWS_ACCESS_KEY_ID=your_access_key_here AWS_SECRET_KEY_ID=your_secret_key_here scarr deploy`

Alternately, if you know how aws credentials files work, scarr supports supports the  `AWS_PROFILE` environment variable as well.

### scarr.yml

The scarr init command will generate a scarr.yml file with pretty much everything you need in it.  You _will_ have to fill out the domainContact details if you want to use scarr for domain registration, though.  The config options are as follows:

- `domain: scarr.io` the domain of your site.
- `name: scarr` used for a number of internal identifiers in the infrastructure, eg the bucket will be called `yourname-bucket`.
- `region: us-west-1` the region to use for everything that's not either region independent (like route53) or that requires a specific region (ACM certs must be in us-east-1 to be used by cloudfront).
- `exclude: ...` a list of regexes (_not_ glob patterns) to exclude from s3 upload.  The regexes will get run against the full relative path of each file (eg if you've got file `/foo/bar/biz/baz` and you run scarr in `bar`, the regex gets run against `biz/baz`.  Note that backslashes need to be escaped in yaml.
  ```
  exclude:
    - "src"
    - "\\.gitignore"
    - "\\.dat$"
  ```
- `domainContact`: the contact info for domain registration.  See the [aws docs](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/domain-register-values-specify.html) for more info.  Most fields are accepted by aws so long as you input _something_, but contactType, countryCode, email, phone, state, and zip all have format validations.
  ```
  domainContact:
    address1: 'fillmein'
    address2: ''
    city: 'fillmein'
    contactType: 'PERSON'
    countryCode: 'US'
    email: 'kevin@kevinkuchta.com'
    firstName: 'fillmein'
    lastName: 'fillmein'
    phoneNumber: '+1.4157582482'
    state: 'CA'
    zipCode: '94117'
  ```


# Commands

### Init

`scarr init -domain example.com -name mycoolproject` generates a directory with a scarr.yml config file in it.  It doesn't touch anything on AWS.

### Deploy

`scarr deploy` should be run in a directory with a scarr.yml file in it.  It checks whether your infrastructure (s3 bucket, cloudfront, etc) is already set up and if not, sets it up.  It then syncs the current directory to S3 and invalidates the cloudfront cache.  TODO: Actually _sync_.  Right now it just copies all files to S3.

- `-skip-setup` skips all the infrastructure setup and just does the S3 sync + cache invalidation.  Scarr won't re-create your infrastructure if it already exists _anyway_, but this option prevents it from even checking the infrastructure, leading to slightly faster file syncs.
- `-auto-register` causes scarr to automatically register the domain (rather than prompting for confirmation from the user) if it's not already in our route53 account and is available to register.
- `-silent` runs scarr without any output except errors and the registration prompt (if -auto-register is off).

# On the code

Let's face it: this codebase is pretty ugly.  The organization is a procedural mess, everything's in the same package, global functions and variables everywhere.  Part of that is because this is literally the first golang code I've ever written, and part of it's because I thought this was going to be a 50-line shell script - I just got carried away and now here we are!  I'll reorganize and clean everything up at some point.

### TODO:
Handle bad input better (eg init with no input gives useless error)
