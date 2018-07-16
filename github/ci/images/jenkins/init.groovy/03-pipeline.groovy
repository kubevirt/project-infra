import jenkins.model.*

import com.cloudbees.hudson.plugins.folder.*
import com.cloudbees.hudson.plugins.folder.properties.*
import com.cloudbees.hudson.plugins.folder.properties.FolderCredentialsProvider.FolderCredentialsProperty
import com.cloudbees.plugins.credentials.impl.*
import com.cloudbees.plugins.credentials.*
import com.cloudbees.plugins.credentials.domains.*
import com.cloudbees.hudson.plugins.folder.computed.DefaultOrphanedItemStrategy

import org.jenkinsci.plugins.github_branch_source.GitHubSCMNavigator
import org.jenkinsci.plugins.github_branch_source.OriginPullRequestDiscoveryTrait
import org.jenkinsci.plugins.github_branch_source.BranchDiscoveryTrait
import org.jenkinsci.plugins.github_branch_source.ForkPullRequestDiscoveryTrait
import org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProjectFactory

import jenkins.branch.OrganizationFolder
import jenkins.scm.api.mixin.ChangeRequestCheckoutStrategy
import jenkins.scm.impl.trait.RegexSCMHeadFilterTrait
import jenkins.scm.impl.trait.RegexSCMSourceFilterTrait


jenkins = Jenkins.instance

def folder = jenkins.getItem("KubeVirt")
if (folder == null) {
    folder = jenkins.createProject(OrganizationFolder, "KubeVirt")
}

// Create GitHub credentials
String id = java.util.UUID.randomUUID().toString()
Credentials c = new UsernamePasswordCredentialsImpl(CredentialsScope.GLOBAL, id, "GitHub access token", "{{ githubUser }}", "{{ githubToken }}")

AbstractFolder<?> folderAbs = AbstractFolder.class.cast(folder)
FolderCredentialsProperty property = folderAbs.getProperties().get(FolderCredentialsProperty.class)

if (property) {
    boolean credExist = false
    for (cred in property.getCredentials()) {
        if (cred.description == c.description) {
            credExist = true
            break
        }
    }
    if (!credExist) {
        property.getStore().addCredentials(Domain.global(), c)
    }
} else {
    property = new FolderCredentialsProperty([c])
    folderAbs.addProperty(property)
}

def navigator = new GitHubSCMNavigator("{{ githubOrg }}")
navigator.credentialsId = c.id
navigator.traits = [
    new RegexSCMSourceFilterTrait("kubevirt"), // build only these repos
    new RegexSCMHeadFilterTrait("master"),     // only inspect branches of this form
    new BranchDiscoveryTrait(2),               // discover all branches, including PRs
    new OriginPullRequestDiscoveryTrait(1),    // Merge
]

folder.navigators.replace(navigator)
folder.description = "KubeVirt repository"
folder.displayName = "KubeVirt"

// Delete orphan items after 5 days
folder.orphanedItemStrategy = new DefaultOrphanedItemStrategy(true, "5", "")

// Configure what Jenkinsfile we should be looking for
WorkflowMultiBranchProjectFactory factory = new WorkflowMultiBranchProjectFactory()
factory.scriptPath = "Jenkinsfile"
folder.projectFactories.replace(factory)

jenkins.save()
