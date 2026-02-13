  // TODO: Implement coverage detection and job generation                                                                                                                                                          
  
package handler

  import (
    "strings"

    "sigs.k8s.io/prow/pkg/github"
  )

  


//Checks for Go file changes in the files list
  func DetectGoFileChanges(files []string) bool {
    for _, file := range files {
      inCoverageDir := strings.HasPrefix(file, "external-plugins/") ||
      strings.HasPrefix(file, "releng/") ||
      strings.HasPrefix(file, "robots/") ||
      strings.HasPrefix(file, "coverage/") ||
      strings.HasPrefix(file, "rehearse/")

      isGoFile :=strings.HasSuffix(file, ".go") ||
                 strings.HasSuffix(file, "go.mod") ||
                 strings.HasSuffix(file, "go.sum") 

      if inCoverageDir && isGoFile {
        return true
      }
    } 
    return false
  }


  func ActOnPrEvent(event *github.PullRequestEvent) bool {
    return event.Action == github.PullRequestActionOpened||
                           event.Action == github.PullRequestActionSynchronize
  }

