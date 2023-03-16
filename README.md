# protonizer

A CLI tool for working with IaC in AWS Proton.

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

## Notes

This is an experimental project and currently only supports generating Proton templates based on Terraform and CodeBuild provisioning.
