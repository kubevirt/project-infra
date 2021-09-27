# TestGrid

[TestGrid](https://testgrid.k8s.io) is an interactive dashboard for viewing tests results in a grid. It parses JUnit reports for generating a grid view from the tests.
There is one dashboard group called `kubevirt` which contains all the KubeVirt dashboards.

TestGrid configuration is stored on a GCS bucket at `gs://kubevirt-prow/testgrid/config`, which is read by testgrid. The job for pushing the config is defined [here](../prow-deploy/files/jobs/kubevirt/project-infra/project-infra-postsubmits.yaml#L757). Testgrid config merger is configured to read KubeVirt's config from the GCS bucket [here](https://github.com/kubernetes/test-infra/blob/master/config/mergelists/prod.yaml).

## Adding new dashboard

The dashboards' configuration is stored in the [gen-config.yaml](https://github.com/kubevirt/project-infra/tree/main/github/ci/testgrid/gen-config.yaml) file. See the example of a dashboard configuration file:
```yaml
dashboards:
  - name: kubevirt_presubmits
  - name: kubevirt_periodics
dashboard_groups:
  - name: kubevirt
    dashboard_names:
      - kubevirt_presubmits
      - kubevirt_periodics
```
Follow these steps to add a new dashboard:

1. Add a new dashboard name in the `dashboards` field inside the dashboards configuration file. **Dashboards must have a dashboard group name as a prefix**.
2. Add the previously added dashboard to the corresponding **dashboard_name** inside the **dashboard_groups** field.

## Adding ProwJob to the TestGrid

After adding a desired dashboard in the `gen-config.yaml` file, add ProwJobs to the dashboard. To do so, define a new **annotations** field in a ProwJob definition. See the example below:
```yaml
annotations:
  testgrid-dashboards: dashboard-name      # [Required] A dashboard already defined in a config.yaml.
  testgrid-tab-name: some-short-name       # [Optional] A shorter name for the tab. If omitted, just uses the job name.
  testgrid-alert-email: me@me.com          # [Optional] An alert email that will be applied to the tab created in the first dashboard specified in testgrid-dashboards.
  description: Words about your job.       # [Optional] A description of your job. If omitted, only the job name is used.
  testgrid-num-columns-recent: "10"        # [Optional] The number of runs in a row that can be omitted before the run is considered stale. The default value is 10.
  testgrid-num-failures-to-alert: "3"      # [Optional] The number of continuous failures before sending an email. The default value is 3.
  testgrid-days-of-results: "15"           # [Optional] The number of days for which the results are visible. The default value is 15.
  testgrid-alert-stale-results-hours: "12" # [Optional] The number of hours that pass with no results after which the email is sent. The default value is 12.
```

The only required field is **testgrid-dashboards**. It must correspond to the name defined in the `gen-config.yaml` file. It is also recommended to add a short description of the job in the **description** field.
The rest of the fields are optional and can be omitted.

If you don't want to include a job on the TestGrid, use this annotation to disable the generation of the TestGrid test group:
```yaml
annotations:
  testgrid-create-test-group: "false"
```

This configuration applies to postsumbit and periodic jobs. Presubmit jobs can be added to the TestGrid, however, if there is no **annnotations** field defined, the job will be omitted in the config file generation.

## Workflow

The process for updating the configuration of KubeVirt's dashboards is automated. Once the changes in dashboards or jobs are merged, a postsumit Prow job uploads the configuration to a GCS bucket. The testgrid instance that runs at https://testgrid.k8s.io will read kubevirt's group config from that bucket, once this is done the latest changes will be live.
