# Kubernetes CRUD role

Kubernetes CRUD role is an high level interface between ansible
and kubernetes operation

It offers a declarative approach on batch deployment
saving thje developer from having to specify common kubernetes
option for every manifest application.

## How does it work ?

To use the role is enough to call with some required variables

    - set_fact:
        manifest_base: /path/to/base/manifest/dir

    - name: Batch operation
      include_role:
        name: kubernetes_crud
      vars:
        # kuberneted-crud vars
        default_kubeconfig_path: /path/to/kubeconfig
        default_namespace: namespace
        default_context: context
        steps: "{{ steps }}"
        # manifest vars
        memory_limit: 512Mi
        
### The steps

Steps is a list of dictionaries that will contains the details of all
operations in the batch.

    steps:
       - desc: A kubernetes operation
         op: apply_manifest
         manifest: relative/path/to/manifest
       - desc: Another kubernetes operation
         op: create_namespace
         namespace: my_namespace

### Implemented operations

- apply_manifest: Applies a manifest
- remove_manifest: Removes a manifest
- create_namespace: Created a namespace
- remove_namespace: Removes a namespace

#### Steps parameters

A single step accepts the following parameters:

- kubeconfig_path: the path to the kubeconfig, if not specified **default_kubeconfig_path** will be used
- op: the operation to perform, if omitted **default_crud_operation** will be used.
- description: A description for the step, if omitted it will be set the same as **op**
- namespace: the namespace to use, if omitted **default_namespace** will be used. For create_namespace, it will be name
 of the namespace to create
- context: the context to use, if omitted **default_context** will be used
- fatal_validation: for apply and remove manifest, specify if the validation has the power to block the operation
, defaults to *true*
- strict_validation: Select if the validation should be strict

Specify to apply/remove manifest

- manifest_inline: Specify a manifest as a YAML string literal
- manifest: Load the manifest from a path relative to **manifest_base**
- manifest_template: Load the manifest from a path relative to **manifest_base** and renders it as jinja2 template
- manifest_url: Load the manifest from a url

## Examples


        steps:
          - desc: Create test namespace
            namespace: test-namespace
            op: create_namespace

          - desc: Apply service from a manifest directly
            manifest: "test-plain.yaml"

          - desc: Apply LimitRange from manifest rendered from template
            manifest_template: "test-template.yml.j2"

          - desc: Apply Service account from invalid remote manifest (and test
            manifest_url: "file://{{ role_path }}/molecule/files/manifests/test-url.yaml"
            fatal_validation: false
            strict_validation: false

          - desc: Apply label to a node from inline manifest
            manifest_inline: |
              apiVersion: v1
              kind: Node
              metadata:
                name: node01
                labels:
                  test_label_key: test-label-value

## Developers info

# How to test ?

This role uses molecule for testing.
Create a python virtualenv and install molecule

then you can launch

    molecule test
    
 The test will launch a kubevirtci instance, test the operation of the role, then shut down
 the instance.
 The command above calls in turn 5 other playbook that can be called singularly to perform testing step by step
 
 * molecule prepare: will launch kubevirtci as test cluster.
 * molecule converge: will invoke the role to perform some operations
 * molecule verify: Will test that operations succeeded correctly
 * molecule cleanup: will revert to a clean state for the test cluster
 * molecule destroy: will shut down kubevirtci test cluster