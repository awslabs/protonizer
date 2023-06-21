## Proton service template

This Proton service template was scaffolded by the [Protonizer CLI tool](https://github.com/awslabs/protonizer).

This service template will be used to create services that will be associated with a Proton environment.


### What's next?

The next step is to design your template's interface.  In other words, how will your consumers interact with your template?  You do this by specifying input and output parameters.

The `input` parameters are defined in your [schema.yaml file](./schema/schema.yaml) using the [standard Open API 3.0 schema specification](https://swagger.io/docs/specification/data-models/).

```yaml
schema:
  format:
    openapi: "3.0.0"
  service_input_type: service
  types:
    service:
      type: object
      description: Service input properties
      properties:

        example_input:
          title: Example Input
          type: string

          description: "This is an example string input"
          default: default
```

The `output` parameters are defined in your [CloudFormation IaC files](infrastructure/cloudformation.yaml) in the `Outputs:` block.

The next step is to author your IaC code using the input parameters provided by Proton.  Make changes in your `instance_infrastructure/cloudformation.yaml` file. You'll use [jinja templating syntax](https://jinja.palletsprojects.com/en/3.1.x/) to access template input parameters.  The example below creates an S3 bucket using the proton input parameter `example_input` as the bucket name, and outputs a parameter `S3BucketArn` with the bucket ARN.

```yaml
Resources:
  S3Bucket:
    Type: 'AWS::S3::Bucket'
    DeletionPolicy: Retain
    Properties:
      BucketName: {{ service_instance.inputs.example_input }}

Outputs:
  S3BucketArn:
    Description: The ARN of the S3 bucket
    Value: !GetAtt S3Bucket.Arn
```


### Publish your template

Once you're happy with how your template looks, you'll need to publish the template to Proton before it can be used.  To publish your template, you can run the following protonizer command.

```
cd my-template/v1
protonizer publish

published my-template:1.0
https://us-east-1.console.aws.amazon.com/proton/home#/templates/services/detail/my-template
```

Note that you'll need to ensure you've set the `publishBucket` key in your `proton.yaml` file.  It should be there if you ran the `new` command using the `--public-bucket` CLI argument.

```yaml
name: my-template
type: service
displayName: my-template
description: A service template scaffolded by the Protonizer CLI tool
publishBucket: my-s3-bucket
compatibleEnvironments:
    - my-env-template:1
```


### Consume your template

Now that your template is published in Proton, you can start creating instances of the template called `services`.  There are a number of ways to do this.

- [Use the GUI console](https://docs.aws.amazon.com/proton/latest/userguide/ag-create-env.html).  Note that if using the approach, Proton can typically generate a custom GUI based on your template's input schema.

- Use the Proton [API](https://docs.aws.amazon.com/proton/latest/APIReference/API_CreateEnvironment.html) (CLI or SDK).  With this approach, you make imperative calls to create environments and services.  For example `aws proton create-environment`.

- [Use Proton service sync](https://docs.aws.amazon.com/proton/latest/userguide/ag-service-sync-configs.html) for a GitOps style workflow.  With this approach, you specify your environments in a YAML file in a Git repo.  You then provide Proton with access to the Git repo that it uses to watch the repo and listen for changes.  When a change is made, Proton will automatically deploy the environments and services.


### Sample Templates

You can find sample Proton templates here.

- [AWS-Managed - CloudFormation](https://github.com/aws-samples/aws-proton-cloudformation-sample-templates)
- [Codebuild - Terraform, CDK, Pulumi, etc.](https://github.com/aws-samples/aws-proton-terraform-sample-templates)
