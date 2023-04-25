# protonizer

A CLI tool for working with IaC in [AWS Proton](https://aws.amazon.com/proton/).

Protonizer allows you take your existing IaC (infrastructure as code) templates and modules and bring them into [AWS Proton](https://aws.amazon.com/proton/) to scale them out across your organization. Proton provides a self-service deployment interface with versioning and traceability for your templates.

Note that this is an experimental project and currently only supports generating Proton templates based on [Terraform](https://www.terraform.io/) and [CodeBuild provisioning](https://docs.aws.amazon.com/proton/latest/userguide/ag-works-prov-methods.html).  The tool also currently only supports primitive [HCL data types](https://developer.hashicorp.com/terraform/language/expressions/types#types) such as `strings`, `numbers`, `bools`, and `lists` of primitive types. This is currently aligned with the Proton schema types that are supported by the Proton console.


## Install

To install the `protonizer` CLI tool, you can download the latest [release](https://github.com/awslabs/protonizer/releases) for your platform and architecture.


## How it works

Protonizer parses your existing Terraform modules and generates Proton [templates](https://docs.aws.amazon.com/proton/latest/userguide/ag-template-authoring.html) with [schemas](https://docs.aws.amazon.com/proton/latest/userguide/ag-schema.html) based on your input and output variables.  It also outputs [manifest.yml](https://docs.aws.amazon.com/proton/latest/userguide/ag-wrap-up.html) files that will run `terraform apply` within a Proton-managed environment.


## Usage

### protonize

The `protonize` command can generate and publish a [CodeBuild provisioning](https://docs.aws.amazon.com/proton/latest/userguide/ag-works-prov-methods.html) template based on an existing Terraform module.

#### Generate a Proton environment

```
protonizer protonize \
  --name my_template \
  --type environment \
  --tool terraform \
  --provisioning codebuild \
  --dir ~/my-existing-tf-module \
  --out ~/proton/templates \
  --bucket my-s3-bucket

template source outputted to ~/proton/templates/my_template
done
```

#### Generate a Proton service (and publish inline)

```
protonizer protonize \
  --name my_template \
  --type service \
  --compatible-env env1:1 --compatible-env env2:1 \
  --tool terraform \
  --provisioning codebuild \
  --dir ~/my-existing-tf-module \
  --out ~/proton/templates \
  --bucket my-s3-bucket \
  --publish

template source outputted to ~/proton/templates/my_template
published my_template:1.0
https://us-east-1.console.aws.amazon.com/proton/home?region=us-east-1#/templates/services/detail/my_template
done
```

### publish

The `publish` command registers and publishes a template with AWS Proton. Just add a `proton.yaml` file to your project and run `protonizer publish`. This is alternative to Proton's [Template sync](https://docs.aws.amazon.com/proton/latest/userguide/ag-template-sync-configs.html) feature, useful for local development or for Git providers that aren't supported.

#### Publish an environment template

proton.yaml

```yaml
name: my_template
type: environment
displayName: My Template
description: "This is my template"
terraformRemoteStateBucket: tf-remote-state-bucket
```

publish using yaml file

```
protonizer publish
published my_template:1.0
https://us-east-1.console.aws.amazon.com/proton/home?region=us-east-1#/templates/environments/detail/my_template
```

#### Publish a service template

proton.yaml

```yaml
name: my_template
type: service
displayName: My Template
description: "This is my template"
terraformRemoteStateBucket: tf-remote-state-bucket
compatibleEnvironments:
  - env1:3
  - env2:4
```

publish using yaml file

```
protonizer publish
published my_template:1.0
https://us-east-1.console.aws.amazon.com/proton/home?region=us-east-1#/templates/services/detail/my_template
```

or specify file name

```
protonizer publish -f file.yml
published my_template:1.0
https://us-east-1.console.aws.amazon.com/proton/home?region=us-east-1#/templates/environments/detail/my_template
```

Note that this can also be done inline with the `protonize --publish` command.


### Terraform variable mapping

To avoid conflicts, if you have variables in your source templates with reserved names in Proton (i.e., `name` and `environment`), they will be removed as template input variables and instead be sourced from proton metadata.


#### Environment templates

If the source terraform module has an input variable named `name`, it will be supplied by the name of the proton environment rather than by template specific input.


#### Service templates

If the source terraform module has a variable named `name`, it will be set to the name of the service and the service instance with a `-` (dash) in between.  If the source terraform module has a variable named `environment`, it will be set to the service instance's environment name.

For example, when creating a service named `sales-api` and a service instance named `dev` associated with a proton environment named `dev`, the Terraform module will get passed the following values:

```hcl
name = "sales-api-dev"
environment = "dev"
```


### Development

#### Setup

- Go 1.19
- Install [pre-commit](https://pre-commit.com/)
- Run `pre-commit install` to setup git hooks

#### Commands

```
 Choose a make command to run

  vet           vet code
  test          run unit tests
  build         build a binary
  autobuild     auto build when source files change
  dockerbuild   build project into a docker container image
  start         build and run local project
  deploy        build code into a container and deploy it to the cloud dev environment
  xplat         multiplatform build
```
