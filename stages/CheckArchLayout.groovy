import com.epam.edp.stages.impl.ci.ProjectType
import com.epam.edp.stages.impl.ci.Stage

@Stage(name = "check-arch", buildTool = "maven", type = ProjectType.APPLICATION)
class CheckArchLayout {
    Script script

    void run(context) {
        script.dir("${context.workDir}") {
            def changes = script.sh(
                    script: 'git diff --diff-filter=ACMR --name-only origin/master',
                    returnStdout: true
            )
            def isTarget = changes.lines().any { it.startsWith("pkg/controller/codebase/chain") }
            if (isTarget) {
                script.httpRequest contentType: 'APPLICATION_JSON',
                        authentication: 'rest-jenkins-gerrit',
                        httpMode: 'PUT',
                        requestBody: '''{"assignee": "pavlo_yemelianov@epam.com"}''',
                        url: "http://gerrit:8080/a/changes/${context.git.changeNumber}/assignee"
            }
        }
    }
}

return CheckArchLayout
