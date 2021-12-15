{
  _config+:: {
    local c = self,
    dashboardNamePrefix: 'Gitea',
    dashboardTags: ['gitea'],
    dashboardPeriod: 'now-1h',
    dashboardTimezone: 'utc',
    dashboardRefresh: '1m',

    // please see https://docs.gitea.io/en-us/config-cheat-sheet/#metrics-metrics
    // Show issue by repository metrics with format gitea_issues_by_repository{repository="org/repo"} 5. Requires Gitea 1.16.0 with ENABLED_ISSUE_BY_REPOSITORY set to true.
    showIssuesByRepository: true,
    // Show graphs for issue by label metrics with format gitea_issues_by_label{label="bug"} 2. Requires Gitea 1.16.0 with ENABLED_ISSUE_BY_LABEL set to true.
    showIssuesByLabel: false,

    // add or remove metrics from dashboard
    giteaStatMetrics: [
      {
        name: 'gitea_organizations',
        description: 'Organizations',
      },
      {
        name: 'gitea_teams',
        description: 'Teams',
      },
      {
        name: 'gitea_users',
        description: 'Users',
      },
      {
        name: 'gitea_repositories',
        description: 'Repositories',
      },
      {
        name: 'gitea_milestones',
        description: 'Milestones',
      },
      {
        name: 'gitea_stars',
        description: 'Stars',
      },
      {
        name: 'gitea_releases',
        description: 'Releases',
      },
      {
        name: 'gitea_issues',
        description: 'Issues',
      },
      {
        name: 'gitea_comments',
        description: 'Comments',
      },
    ],
    //set this for using label colors on graphs
    issueLabels: [
      {
        label: 'bug',
        color: '#ee0701',
      },
      {
        label: 'duplicate',
        color: '#cccccc',
      },
      {
        label: 'invalid',
        color: '#e6e6e6',
      },
      {
        label: 'enhancement',
        color: '#84b6eb',
      },
      {
        label: 'help wanted',
        color: '#128a0c',
      },
      {
        label: 'question',
        color: '#cc317c',
      },
    ],
  },
}
