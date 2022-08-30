module.exports = {
    branchPrefix: 'test-renovate/',
    dryRun: false,
    username: 'renovate-release',
    gitAuthor: 'Renovate Bot <bot@renovateapp.com>',
    onboarding: false,
    platform: 'github',
    includeForks: false,
    repositories: [
        'hikhvar/mqtt2prometheus',
    ],
    packageRules: [
        {
            description: 'lockFileMaintenance',
            matchUpdateTypes: [
                'pin',
                'digest',
                'patch',
                'minor',
                'major',
                'lockFileMaintenance',
            ],
            dependencyDashboardApproval: false,
            stabilityDays: 10,
        },
    ],
};
