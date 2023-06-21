## Proton environment template

This Proton environment template was scaffolded by the [Protonizer CLI tool](https://github.com/awslabs/protonizer).

This environment template will be used to create shared infrastructure associated with one more many service templates.


### What's next?

The next step is to design your template's interface.  In other words, how will your consumers interact with your template?  You do this by specifying input and output parameters.

The `input` parameters are defined in your [schema.yaml file](./schema/schema.yaml) using the [standard Open API 3.0 schema specification](https://swagger.io/docs/specification/data-models/).

```yaml
schema:
  format:
    openapi: "3.0.0"
  environment_input_type: environment
  types:
    environment:
      type: object
      description: Environment input properties
      properties:

        # define your input properties here
        example_input:
          title: Example Input
          type: string
          description: "This is an example string input"
          default: default
```

The `output` parameters are defined in the generated [infrastructure/outputs.tf](./infrastructure/outputs.tf) file.  The generated [output.sh](./infrastructure/output.sh) script will read your Terraform output variables and send them to Proton as outputs.

The next step is to author your IaC code using the input parameters provided by Proton.  Make changes to the `.tf` files in the [infrastructure](./infrastructure) directory.  The Proton input parameters are passed in to Terraform using standard input variables.  The example below creates a CloudWatch log group bucket using the proton input parameter `example_input` as the name, and outputs a parameter `LogGroupName` with the log group ARN.

main.tf
```hcl
provider "aws" {
  default_tags {
    tags = {
      "proton:environment" = var.environment.name
    }
  }
}

resource "aws_cloudwatch_log_group" "example" {
  name = var.environment.inputs.example_input
}
```

outputs.tf
```hcl
output "LogGroupArn" {
  description = "the s3 bucket that was created"
  value       = aws_cloudwatch_log_group.example.arn
}
```


### Publish your template

Once you're happy with how your template looks, you'll need to publish the template to Proton before it can be used.  To publish your template, you can run the following protonizer command.

```
cd my-template/v1
protonizer publish

published my-template:1.0
https://us-east-1.console.aws.amazon.com/proton/home#/templates/environments/detail/my-template
```

Note that you'll need to ensure you've set the `publishBucket` key in your `proton.yaml` file.  It should be there if you ran the `new` command using the `--public-bucket` CLI argument.

```yaml
name: my-template
type: environment
displayName: my-template
description: An environment template scaffolded by the Protonizer CLI tool
publishBucket: my-s3-bucket
```


### Consume your template

Now that your template is published in Proton, you can start creating instances of the template called `environments`.  There are a number of ways to do this.

- [Use the GUI console](https://docs.aws.amazon.com/proton/latest/userguide/ag-create-env.html).  Note that if using the approach, Proton can typically generate a custom GUI based on your template's input schema.

- Use the Proton [API](https://docs.aws.amazon.com/proton/latest/APIReference/API_CreateEnvironment.html) (CLI or SDK).  With this approach, you make imperative calls to create environments and services.  For example `aws proton create-environment`.

- [Use Proton service sync](https://docs.aws.amazon.com/proton/latest/userguide/ag-service-sync-configs.html) for a GitOps style workflow.  With this approach, you specify your environments in a YAML file in a Git repo.  You then provide Proton with access to the Git repo that it uses to watch the repo and listen for changes.  When a change is made, Proton will automatically deploy the environments and services.


### Sample Templates

You can find sample Proton templates here.

- [AWS-Managed - CloudFormation](https://github.com/aws-samples/aws-proton-cloudformation-sample-templates)
- [Codebuild - Terraform, CDK, Pulumi, etc.](https://github.com/aws-samples/aws-proton-terraform-sample-templates)
