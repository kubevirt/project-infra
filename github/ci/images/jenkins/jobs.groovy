multibranchPipelineJob("KubeVirt") {
    displayName "KubeVirt"
    description "KubeVirt repository project"
    branchSources {
        github {
            repoOwner("{{ githubOrg }}")
            repository("kubevirt")
            scanCredentialsId("github-token")
            includes("master")
        }
    }
    orphanedItemStrategy {
        discardOldItems {
            numToKeep(20)
        }
    }
}
