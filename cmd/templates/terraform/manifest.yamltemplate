infrastructure:
  templates:
    - rendering_engine: codebuild
      settings:
        image: aws/codebuild/standard:6.0
        runtimes:
          golang: 1.18 # not needed, but required by proton (for now)
        env:
          variables:
            TF_VERSION: 1.4.5
            AWS_REGION: us-east-1
            TF_STATE_BUCKET: {{ .TerraformS3StateBucket }}

        provision:

          # get proton metadata from input file
          - export IN=$(cat proton-inputs.json) && echo ${IN}
          - export PROTON_ENV=$(echo $IN | jq '.environment.name' -r)
{{ if eq .TemplateType "service" }}
          - export PROTON_SVC=$(echo $IN | jq '.service.name' -r)
          - export PROTON_SVC_INSTANCE=$(echo $IN | jq '.service_instance.name' -r)
{{ end }}
          # set terraform remote state bucket key
{{ if eq .TemplateType "service" }}
          - export KEY=svc.{{.TemplateName}}.${PROTON_ENV}.${PROTON_SVC}.${PROTON_SVC_INSTANCE}
{{ else }}
          - export KEY=env.{{.TemplateName}}.${PROTON_ENV}
{{ end }}
          - echo "remote state = ${TF_STATE_BUCKET}/${KEY}"

          # install terraform cli
          - echo "Installing Terraform CLI ${TF_VERSION}"
          - chmod +x ./install-terraform.sh && ./install-terraform.sh ${TF_VERSION}

          # provision, storing state in an s3 bucket
          - terraform init -backend-config="bucket=${TF_STATE_BUCKET}" -backend-config="key=${KEY}.tfstate"
          - terraform apply -var-file=proton-inputs.json -auto-approve

          # pass terraform output to proton
          - chmod +x ./output.sh && ./output.sh

        deprovision:

           # get proton metadata from input file
          - export IN=$(cat proton-inputs.json) && echo ${IN}
          - export PROTON_ENV=$(echo $IN | jq '.environment.name' -r)
{{ if eq .TemplateType "service" }}
          - export PROTON_SVC=$(echo $IN | jq '.service.name' -r)
          - export PROTON_SVC_INSTANCE=$(echo $IN | jq '.service_instance.name' -r)
{{ end }}
          # set terraform remote state bucket key
 {{ if eq .TemplateType "service" }}
          - export KEY=svc.{{.TemplateName}}.${PROTON_ENV}.${PROTON_SVC}.${PROTON_SVC_INSTANCE}
{{ else }}
          - export KEY=env.{{.TemplateName}}.${PROTON_ENV}
{{ end }}
          - echo "remote state = ${TF_STATE_BUCKET}/${KEY}"

          # install terraform cli
          - echo "Installing Terraform CLI ${TF_VERSION}"
          - chmod +x ./install-terraform.sh && ./install-terraform.sh ${TF_VERSION}

          # destroy environment
          - terraform init -backend-config="bucket=${TF_STATE_BUCKET}" -backend-config="key=${KEY}.tfstate"
          - terraform destroy -var-file=proton-inputs.json -auto-approve
