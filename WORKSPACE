load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "3efbd23e195727a67f87b2a04fb4388cc7a11a0c0c2cf33eec225fb8ffbb27ea",
    strip_prefix = "rules_docker-0.14.2",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.14.2/rules_docker-v0.14.2.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "87f0fb9747854cb76a0a82430adccb6269f7d394237104a4523b51061c469171",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.1/rules_go-v0.23.1.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.1/rules_go-v0.23.1.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "bfd86b3cbe855d6c16c6fce60d76bd51f5c8dbc9cfcaef7a2bb5c1aafd0710e8",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.0/bazel-gazelle-v0.21.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.0/bazel-gazelle-v0.21.0.tar.gz",
    ],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "069617be63d72c2c70a14d832102a98d65fe12e02661f7c42797bc390c80017f",
    strip_prefix = "buildtools-2.2.0",
    url = "https://github.com/bazelbuild/buildtools/archive/2.2.0.tar.gz",
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "9748c0d90e54ea09e5e75fb7fac16edce15d2028d4356f32211cfa3c0e956564",
    strip_prefix = "protobuf-3.11.4",
    urls = ["https://github.com/protocolbuffers/protobuf/archive/v3.11.4.zip"],
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.13.4",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

container_pull(
    name = "infra-base",
    registry = "gcr.io",
    repository = "k8s-testimages/bootstrap",
    tag = "v20190516-c6832d9",
)

container_pull(
    name = "release-tool-base",
    registry = "index.docker.io",
    repository = "kubevirtci/release-tool-base",
    tag = "v20210120-b86882c9314933ba1a0c77965ed9d54a747f7957",
)

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    sum = "h1:zWTV+LMdc3kaiJMSTOFz2UgSBgx8RNQoTGiZu3fR9S0=",
    version = "v1.32.0",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:KrsHThm5nFk34YtATK1LsThyGhGbGe1olrte/HInHvs=",
    version = "v0.0.0-20210326060303-6b1517762897",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:cokOdA+Jmi5PJGXLlLllQSgYigAEfHXJAERHVMaCc2k=",
    version = "v0.3.3",
)

rules_gitops_version = "8d9416a36904c537da550c95dc7211406b431db9"

http_archive(
    name = "com_adobe_rules_gitops",
    sha256 = "25601ed932bab631e7004731cf81a40bd00c9a34b87c7de35f6bc905c37ef30d",
    strip_prefix = "rules_gitops-%s" % rules_gitops_version,
    urls = ["https://github.com/adobe/rules_gitops/archive/%s.zip" % rules_gitops_version],
)

load("@com_adobe_rules_gitops//gitops:deps.bzl", "rules_gitops_dependencies")

rules_gitops_dependencies()

load("@com_adobe_rules_gitops//gitops:repositories.bzl", "rules_gitops_repositories")

rules_gitops_repositories()

go_repository(
    name = "ag_pack_amqp",
    importpath = "pack.ag/amqp",
    sum = "h1:cuNDWLUTbKRtEZwhB0WQBXf9pGbm87pUBXQhvcFxBWg=",
    version = "v0.11.2",
)

go_repository(
    name = "cc_mvdan_interfacer",
    importpath = "mvdan.cc/interfacer",
    sum = "h1:WX1yoOaKQfddO/mLzdV4wptyWgoH/6hwLs7QHTixo0I=",
    version = "v0.0.0-20180901003855-c20040233aed",
)

go_repository(
    name = "cc_mvdan_lint",
    importpath = "mvdan.cc/lint",
    sum = "h1:DxJ5nJdkhDlLok9K6qO+5290kphDJbHOQO1DFFFTeBo=",
    version = "v0.0.0-20170908181259-adc824a0674b",
)

go_repository(
    name = "cc_mvdan_unparam",
    importpath = "mvdan.cc/unparam",
    sum = "h1:kAREL6MPwpsk1/PQPFD3Eg7WAQR5mPTWZJaBiG5LDbY=",
    version = "v0.0.0-20200501210554-b37ab49443f7",
)

go_repository(
    name = "cc_mvdan_xurls_v2",
    importpath = "mvdan.cc/xurls/v2",
    sum = "h1:r1zSOSNS/kqtpmATyMMMvaZ4/djsesbYz5kr0+qMRWc=",
    version = "v2.0.0",
)

go_repository(
    name = "co_honnef_go_tools",
    importpath = "honnef.co/go/tools",
    sum = "h1:AWNL1W1i7f0wNZ8VwOKNJ0sliKvOF/adn0EHenfUh+c=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_agnivade_levenshtein",
    importpath = "github.com/agnivade/levenshtein",
    sum = "h1:3oJU7J3FGFmyhn8KHjmVaZCN5hxTr7GxgRue+sxIXdQ=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_alcortesm_tgz",
    importpath = "github.com/alcortesm/tgz",
    sum = "h1:uSoVVbwJiQipAclBbw+8quDsfcvFjOpI5iCf4p/cqCs=",
    version = "v0.0.0-20161220082320-9c5fe88206d7",
)

go_repository(
    name = "com_github_alecthomas_kingpin",
    importpath = "github.com/alecthomas/kingpin",
    sum = "h1:5svnBTFgJjZvGKyYBtMB0+m5wvrbUHiqye8wRJMlnYI=",
    version = "v2.2.6+incompatible",
)

go_repository(
    name = "com_github_alecthomas_template",
    importpath = "github.com/alecthomas/template",
    sum = "h1:JYp7IbQjafoB+tBA3gMyHYHrpOtNuDiK/uB5uXxq5wM=",
    version = "v0.0.0-20190718012654-fb15b899a751",
)

go_repository(
    name = "com_github_alecthomas_units",
    importpath = "github.com/alecthomas/units",
    sum = "h1:UQZhZ2O0vMHr2cI+DC1Mbh0TJxzA3RcLoMsFw+aXw7E=",
    version = "v0.0.0-20190924025748-f65c72e2690d",
)

go_repository(
    name = "com_github_andreyvit_diff",
    importpath = "github.com/andreyvit/diff",
    sum = "h1:bvNMNQO63//z+xNgfBlViaCIJKLlCJ6/fmUseuG0wVQ=",
    version = "v0.0.0-20170406064948-c7f18ee00883",
)

go_repository(
    name = "com_github_andybalholm_brotli",
    importpath = "github.com/andybalholm/brotli",
    sum = "h1:bZ28Hqta7TFAK3Q08CMvv8y3/8ATaEqv2nGoc6yff6c=",
    version = "v0.0.0-20190621154722-5f990b63d2d6",
)

go_repository(
    name = "com_github_andygrunwald_go_gerrit",
    importpath = "github.com/andygrunwald/go-gerrit",
    sum = "h1:uUuUZipfD5nPl2L/i0I3N4iRKJcoO2CPjktaH/kP9gQ=",
    version = "v0.0.0-20190120104749-174420ebee6c",
)

go_repository(
    name = "com_github_andygrunwald_go_jira",
    importpath = "github.com/andygrunwald/go-jira",
    sum = "h1:vvIImGgX32bHfoiyUwkNo+/YrPnRczNarvhLOncP6dE=",
    version = "v1.13.0",
)

go_repository(
    name = "com_github_anmitsu_go_shlex",
    importpath = "github.com/anmitsu/go-shlex",
    sum = "h1:kFOfPq6dUM1hTo4JG6LR5AXSUEsOjtdm0kw0FtQtMJA=",
    version = "v0.0.0-20161002113705-648efa622239",
)

go_repository(
    name = "com_github_antihax_optional",
    importpath = "github.com/antihax/optional",
    sum = "h1:uZuxRZCz65cG1o6K/xUqImNcYKtmk9ylqaH0itMSvzA=",
    version = "v0.0.0-20180407024304-ca021399b1a6",
)

go_repository(
    name = "com_github_apache_thrift",
    importpath = "github.com/apache/thrift",
    sum = "h1:pODnxUFNcjP9UTLZGTdeh+j16A8lJbRvD3rOtrk/7bs=",
    version = "v0.12.0",
)

go_repository(
    name = "com_github_apex_log",
    importpath = "github.com/apex/log",
    sum = "h1:1fyfbPvUwD10nMoh3hY6MXzvZShJQn9/ck7ATgAt5pA=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_apex_logs",
    importpath = "github.com/apex/logs",
    sum = "h1:KmEBVwfDUOTFcBO8cfkJYwdQ5487UZSN+GteOGPmiro=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_aphistic_golf",
    importpath = "github.com/aphistic/golf",
    sum = "h1:2KLQMJ8msqoPHIPDufkxVcoTtcmE5+1sL9950m4R9Pk=",
    version = "v0.0.0-20180712155816-02c07f170c5a",
)

go_repository(
    name = "com_github_aphistic_sweet",
    importpath = "github.com/aphistic/sweet",
    sum = "h1:I4z+fAUqvKfvZV/CHi5dV0QuwbmIvYYFDjG0Ss5QpAs=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_armon_circbuf",
    importpath = "github.com/armon/circbuf",
    sum = "h1:QEF07wC0T1rKkctt1RINW/+RMTVmiwxETico2l3gxJA=",
    version = "v0.0.0-20150827004946-bbbad097214e",
)

go_repository(
    name = "com_github_armon_consul_api",
    importpath = "github.com/armon/consul-api",
    sum = "h1:G1bPvciwNyF7IUmKXNt9Ak3m6u9DE1rF+RmtIkBpVdA=",
    version = "v0.0.0-20180202201655-eb2c6b5be1b6",
)

go_repository(
    name = "com_github_armon_go_metrics",
    importpath = "github.com/armon/go-metrics",
    sum = "h1:8GUt8eRujhVEGZFFEjBj46YV4rDjvGrNxb0KMWYkL2I=",
    version = "v0.0.0-20180917152333-f0300d1749da",
)

go_repository(
    name = "com_github_armon_go_radix",
    importpath = "github.com/armon/go-radix",
    sum = "h1:BUAU3CGlLvorLI26FmByPp2eC2qla6E1Tw+scpcg/to=",
    version = "v0.0.0-20180808171621-7fddfc383310",
)

go_repository(
    name = "com_github_armon_go_socks5",
    importpath = "github.com/armon/go-socks5",
    sum = "h1:0CwZNZbxp69SHPdPJAN/hZIm0C4OItdklCFmMRWYpio=",
    version = "v0.0.0-20160902184237-e75332964ef5",
)

go_repository(
    name = "com_github_asaskevich_govalidator",
    importpath = "github.com/asaskevich/govalidator",
    sum = "h1:zV3ejI06GQ59hwDQAvmK1qxOQGB3WuVTRoY0okPTAv0=",
    version = "v0.0.0-20200108200545-475eaeb16496",
)

go_repository(
    name = "com_github_aws_aws_k8s_tester",
    importpath = "github.com/aws/aws-k8s-tester",
    sum = "h1:Zr5NWiRK5fhmRIlhrsTwrY8yB488FyN6iulci2D7VaI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_aws_aws_sdk_go",
    importpath = "github.com/aws/aws-sdk-go",
    sum = "h1:izATc/E0+HcT5YHmaQVjn7GHCoqaBxn0PGo6Zq5UNFA=",
    version = "v1.34.30",
)

go_repository(
    name = "com_github_aybabtme_rgbterm",
    importpath = "github.com/aybabtme/rgbterm",
    sum = "h1:WWB576BN5zNSZc/M9d/10pqEx5VHNhaQ/yOVAkmj5Yo=",
    version = "v0.0.0-20170906152045-cc83f3b3ce59",
)

go_repository(
    name = "com_github_azure_azure_amqp_common_go_v2",
    importpath = "github.com/Azure/azure-amqp-common-go/v2",
    sum = "h1:+QbFgmWCnPzdaRMfsI0Yb6GrRdBj5jVL8N3EXuEUcBQ=",
    version = "v2.1.0",
)

go_repository(
    name = "com_github_azure_azure_pipeline_go",
    importpath = "github.com/Azure/azure-pipeline-go",
    sum = "h1:6oiIS9yaG6XCCzhgAgKFfIWyo4LLCiDhZot6ltoThhY=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_azure_azure_sdk_for_go",
    importpath = "github.com/Azure/azure-sdk-for-go",
    sum = "h1:m4oQOm3HXtQh2Ipata+pLSS1kGUD/7ikkvNq81XM/7s=",
    version = "v46.3.0+incompatible",
)

go_repository(
    name = "com_github_azure_azure_service_bus_go",
    importpath = "github.com/Azure/azure-service-bus-go",
    sum = "h1:G1qBLQvHCFDv9pcpgwgFkspzvnGknJRR0PYJ9ytY/JA=",
    version = "v0.9.1",
)

go_repository(
    name = "com_github_azure_azure_storage_blob_go",
    importpath = "github.com/Azure/azure-storage-blob-go",
    sum = "h1:53qhf0Oxa0nOjgbDeeYPUeyiNmafAFEY95rZLK0Tj6o=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_azure_go_ansiterm",
    importpath = "github.com/Azure/go-ansiterm",
    sum = "h1:w+iIsaOQNcT7OZ575w+acHgRric5iCyQh+xv+KJ4HB8=",
    version = "v0.0.0-20170929234023-d6e3b3328b78",
)

go_repository(
    name = "com_github_azure_go_autorest",
    importpath = "github.com/Azure/go-autorest",
    replace = "github.com/Azure/go-autorest",
    sum = "h1:V5VMDjClD3GiElqLWO7mz2MxNAK/vTfRHdAubSIPRgs=",
    version = "v14.2.0+incompatible",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest",
    importpath = "github.com/Azure/go-autorest/autorest",
    sum = "h1:LIzfhNo9I3+il0KO2JY1/lgJmjig7lY0wFulQNZkbtg=",
    version = "v0.11.6",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_adal",
    importpath = "github.com/Azure/go-autorest/autorest/adal",
    sum = "h1:1/DtH4Szusk4psLBrJn/gocMRIf1ji30WAz3GfyULRQ=",
    version = "v0.9.4",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_azure_auth",
    importpath = "github.com/Azure/go-autorest/autorest/azure/auth",
    sum = "h1:iM6UAvjR97ZIeR93qTcwpKNMpV+/FTWjwEbuPD495Tk=",
    version = "v0.4.2",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_azure_cli",
    importpath = "github.com/Azure/go-autorest/autorest/azure/cli",
    sum = "h1:LXl088ZQlP0SBppGFsRZonW6hSvwgL5gRByMbvUbx8U=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_date",
    importpath = "github.com/Azure/go-autorest/autorest/date",
    sum = "h1:7gUk1U5M/CQbp9WoqinNzJar+8KY+LPI6wiWrP/myHw=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_mocks",
    importpath = "github.com/Azure/go-autorest/autorest/mocks",
    sum = "h1:K0laFcLE6VLTOwNgSxaGbUcLPuGXlNkbVvq4cW4nIHk=",
    version = "v0.4.1",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_to",
    importpath = "github.com/Azure/go-autorest/autorest/to",
    sum = "h1:oXVqrxakqqV1UZdSazDOPOLvOIz+XA683u8EctwboHk=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_azure_go_autorest_autorest_validation",
    importpath = "github.com/Azure/go-autorest/autorest/validation",
    sum = "h1:3I9AAI63HfcLtphd9g39ruUwRI+Ca+z/f36KHPFRUss=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_azure_go_autorest_logger",
    importpath = "github.com/Azure/go-autorest/logger",
    sum = "h1:e4RVHVZKC5p6UANLJHkM4OfR1UKZPj8Wt8Pcx+3oqrE=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_azure_go_autorest_tracing",
    importpath = "github.com/Azure/go-autorest/tracing",
    sum = "h1:TYi4+3m5t6K48TGI9AUdb+IzbnSxvnvUMfuitfgcfuo=",
    version = "v0.6.0",
)

go_repository(
    name = "com_github_beorn7_perks",
    importpath = "github.com/beorn7/perks",
    sum = "h1:VlbKKnNfV8bJzeqoa4cOKqO6bYr3WgKZxO8Z16+hsOM=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_bgentry_speakeasy",
    importpath = "github.com/bgentry/speakeasy",
    sum = "h1:ByYyxL9InA1OWqxJqqp2A5pYHUrCiAL6K3J+LKSsQkY=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_bitly_go_simplejson",
    importpath = "github.com/bitly/go-simplejson",
    sum = "h1:6IH+V8/tVMab511d5bn4M7EwGXZf9Hj6i2xSwkNEM+Y=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_bketelsen_crypt",
    importpath = "github.com/bketelsen/crypt",
    sum = "h1:+0HFd5KSZ/mm3JmhmrDukiId5iR6w4+BdFtfSy4yWIc=",
    version = "v0.0.3-0.20200106085610-5cbc8cc4026c",
)

go_repository(
    name = "com_github_blakesmith_ar",
    importpath = "github.com/blakesmith/ar",
    sum = "h1:m935MPodAbYS46DG4pJSv7WO+VECIWUQ7OJYSoTrMh4=",
    version = "v0.0.0-20190502131153-809d4375e1fb",
)

go_repository(
    name = "com_github_blang_semver",
    importpath = "github.com/blang/semver",
    sum = "h1:cQNTCjp13qL8KC3Nbxr/y2Bqb63oX6wdnnjpJbkM4JQ=",
    version = "v3.5.1+incompatible",
)

go_repository(
    name = "com_github_bmizerany_assert",
    importpath = "github.com/bmizerany/assert",
    sum = "h1:DDGfHa7BWjL4YnC6+E63dPcxHo2sUxDIu8g3QgEJdRY=",
    version = "v0.0.0-20160611221934-b7ed37b82869",
)

go_repository(
    name = "com_github_bmizerany_perks",
    importpath = "github.com/bmizerany/perks",
    sum = "h1:AP/Y7sqYicnjGDfD5VcY4CIfh1hRXBUavxrvELjTiOE=",
    version = "v0.0.0-20141205001514-d9a9656a3a4b",
)

go_repository(
    name = "com_github_bombsimon_wsl_v2",
    importpath = "github.com/bombsimon/wsl/v2",
    sum = "h1:/DdSteYCq4lPX+LqDg7mdoxm14UxzZPoDT0taYc3DTU=",
    version = "v2.2.0",
)

go_repository(
    name = "com_github_bombsimon_wsl_v3",
    importpath = "github.com/bombsimon/wsl/v3",
    sum = "h1:E5SRssoBgtVFPcYWUOFJEcgaySgdtTNYzsSKDOY7ss8=",
    version = "v3.1.0",
)

go_repository(
    name = "com_github_bshuster_repo_logrus_logstash_hook",
    importpath = "github.com/bshuster-repo/logrus-logstash-hook",
    sum = "h1:pgAtgj+A31JBVtEHu2uHuEx0n+2ukqUJnS2vVe5pQNA=",
    version = "v0.4.1",
)

go_repository(
    name = "com_github_bugsnag_bugsnag_go",
    importpath = "github.com/bugsnag/bugsnag-go",
    sum = "h1:rFt+Y/IK1aEZkEHchZRSq9OQbsSzIT/OrI8YFFmRIng=",
    version = "v0.0.0-20141110184014-b1d153021fcd",
)

go_repository(
    name = "com_github_bugsnag_osext",
    importpath = "github.com/bugsnag/osext",
    sum = "h1:otBG+dV+YK+Soembjv71DPz3uX/V/6MMlSyD9JBQ6kQ=",
    version = "v0.0.0-20130617224835-0dd3f918b21b",
)

go_repository(
    name = "com_github_bugsnag_panicwrap",
    importpath = "github.com/bugsnag/panicwrap",
    sum = "h1:nvj0OLI3YqYXer/kZD8Ri1aaunCxIEsOst1BVJswV0o=",
    version = "v0.0.0-20151223152923-e2c28503fcd0",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_burntsushi_xgb",
    importpath = "github.com/BurntSushi/xgb",
    sum = "h1:1BDTz0u9nC3//pOCMdNH+CiXJVYJh5UQNCOBG7jbELc=",
    version = "v0.0.0-20160522181843-27f122750802",
)

go_repository(
    name = "com_github_bwmarrin_snowflake",
    importpath = "github.com/bwmarrin/snowflake",
    sum = "h1:dRbqXFjM10uA3wdrVZ8Kh19uhciRMOroUYJ7qAqDLhY=",
    version = "v0.0.0",
)

go_repository(
    name = "com_github_caarlos0_ctrlc",
    importpath = "github.com/caarlos0/ctrlc",
    sum = "h1:2DtF8GSIcajgffDFJzyG15vO+1PuBWOMUdFut7NnXhw=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_campoy_unique",
    importpath = "github.com/campoy/unique",
    sum = "h1:V9a67dfYqPLAvzk5hMQOXYJlZ4SLIXgyKIE+ZiHzgGQ=",
    version = "v0.0.0-20180121183637-88950e537e7e",
)

go_repository(
    name = "com_github_cavaliercoder_go_cpio",
    importpath = "github.com/cavaliercoder/go-cpio",
    sum = "h1:hHg27A0RSSp2Om9lubZpiMgVbvn39bsUmW9U5h0twqc=",
    version = "v0.0.0-20180626203310-925f9528c45e",
)

go_repository(
    name = "com_github_census_instrumentation_opencensus_proto",
    importpath = "github.com/census-instrumentation/opencensus-proto",
    sum = "h1:glEXhBS5PSLLv4IXzLA5yPRVX4bilULVyxxbrfOtDAk=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_cespare_xxhash",
    importpath = "github.com/cespare/xxhash",
    sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_cespare_xxhash_v2",
    importpath = "github.com/cespare/xxhash/v2",
    sum = "h1:6MnRN8NT7+YBpUIWxHtefFZOKTAPgGjpQSxqLNn0+qY=",
    version = "v2.1.1",
)

go_repository(
    name = "com_github_chai2010_gettext_go",
    importpath = "github.com/chai2010/gettext-go",
    sum = "h1:7aWHqerlJ41y6FOsEUvknqgXnGmJyJSbjhAWq5pO4F8=",
    version = "v0.0.0-20160711120539-c6fed771bfd5",
)

go_repository(
    name = "com_github_chzyer_logex",
    importpath = "github.com/chzyer/logex",
    sum = "h1:Swpa1K6QvQznwJRcfTfQJmTE72DqScAa40E+fbHEXEE=",
    version = "v1.1.10",
)

go_repository(
    name = "com_github_chzyer_readline",
    importpath = "github.com/chzyer/readline",
    sum = "h1:fY5BOSpyZCqRo5OhCuC+XN+r/bBCmeuuJtjz+bCNIf8=",
    version = "v0.0.0-20180603132655-2972be24d48e",
)

go_repository(
    name = "com_github_chzyer_test",
    importpath = "github.com/chzyer/test",
    sum = "h1:q763qf9huN11kDQavWsoZXJNW3xEE4JJyHa5Q25/sd8=",
    version = "v0.0.0-20180213035817-a1ea475d72b1",
)

go_repository(
    name = "com_github_cihub_seelog",
    importpath = "github.com/cihub/seelog",
    sum = "h1:kHaBemcxl8o/pQ5VM1c8PVE1PubbNx3mjUr09OqWGCs=",
    version = "v0.0.0-20170130134532-f561c5e57575",
)

go_repository(
    name = "com_github_clarketm_json",
    importpath = "github.com/clarketm/json",
    sum = "h1:0JketcMdLC16WGnRGJiNmTXuQznDEQaiknxSPRBxg+k=",
    version = "v1.13.4",
)

go_repository(
    name = "com_github_client9_misspell",
    importpath = "github.com/client9/misspell",
    sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
    version = "v0.3.4",
)

go_repository(
    name = "com_github_cloudevents_sdk_go",
    importpath = "github.com/cloudevents/sdk-go",
    sum = "h1:gS5I0s2qPmdc4GBPlUmzZU7RH30BaiOdcRJ1RkXnPrc=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_cloudevents_sdk_go_v2",
    importpath = "github.com/cloudevents/sdk-go/v2",
    sum = "h1:AUdGJwaSUnA+VvepKqgjy6XDkPcf0hf/3L7icEs1ibs=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_cloudflare_cloudflare_go",
    importpath = "github.com/cloudflare/cloudflare-go",
    sum = "h1:bhMGoNhAg21DuqJjU9jQepRRft6vYfo6pejT3NN4V6A=",
    version = "v0.13.2",
)

go_repository(
    name = "com_github_cncf_udpa_go",
    importpath = "github.com/cncf/udpa/go",
    sum = "h1:WBZRG4aNOuI15bLRrCgN8fCq8E5Xuty6jGbmSNEvSsU=",
    version = "v0.0.0-20191209042840-269d4d468f6f",
)

go_repository(
    name = "com_github_cockroachdb_datadriven",
    importpath = "github.com/cockroachdb/datadriven",
    sum = "h1:OaNxuTZr7kxeODyLWsRMC+OD03aFUH+mW6r2d+MWa5Y=",
    version = "v0.0.0-20190809214429-80d97fb3cbaa",
)

go_repository(
    name = "com_github_containerd_cgroups",
    importpath = "github.com/containerd/cgroups",
    sum = "h1:tSNMc+rJDfmYntojat8lljbt1mgKNpTxUZJsSzJ9Y1s=",
    version = "v0.0.0-20190919134610-bf292b21730f",
)

go_repository(
    name = "com_github_containerd_console",
    importpath = "github.com/containerd/console",
    sum = "h1:uict5mhHFTzKLUCufdSLym7z/J0CbBJT59lYbP9wtbg=",
    version = "v0.0.0-20180822173158-c12b1e7919c1",
)

go_repository(
    name = "com_github_containerd_containerd",
    importpath = "github.com/containerd/containerd",
    sum = "h1:LoIzb5y9x5l8VKAlyrbusNPXqBY0+kviRloxFUMFwKc=",
    version = "v1.3.3",
)

go_repository(
    name = "com_github_containerd_continuity",
    importpath = "github.com/containerd/continuity",
    sum = "h1:kIFnQBO7rQ0XkMe6xEwbybYHBEaWmh/f++laI6Emt7M=",
    version = "v0.0.0-20200107194136-26c1120b8d41",
)

go_repository(
    name = "com_github_containerd_fifo",
    importpath = "github.com/containerd/fifo",
    sum = "h1:PUD50EuOMkXVcpBIA/R95d56duJR9VxhwncsFbNnxW4=",
    version = "v0.0.0-20190226154929-a9fb20d87448",
)

go_repository(
    name = "com_github_containerd_go_runc",
    importpath = "github.com/containerd/go-runc",
    sum = "h1:esQOJREg8nw8aXj6uCN5dfW5cKUBiEJ/+nni1Q/D/sw=",
    version = "v0.0.0-20180907222934-5a6d9f37cfa3",
)

go_repository(
    name = "com_github_containerd_ttrpc",
    importpath = "github.com/containerd/ttrpc",
    sum = "h1:dlfGmNcE3jDAecLqwKPMNX6nk2qh1c1Vg1/YTzpOOF4=",
    version = "v0.0.0-20190828154514-0e0f228740de",
)

go_repository(
    name = "com_github_containerd_typeurl",
    importpath = "github.com/containerd/typeurl",
    sum = "h1:JNn81o/xG+8NEo3bC/vx9pbi/g2WI8mtP2/nXzu297Y=",
    version = "v0.0.0-20180627222232-a93fcdb778cd",
)

go_repository(
    name = "com_github_coreos_bbolt",
    importpath = "github.com/coreos/bbolt",
    sum = "h1:n6AiVyVRKQFNb6mJlwESEvvLoDyiTzXX7ORAUlkeBdY=",
    version = "v1.3.3",
)

go_repository(
    name = "com_github_coreos_etcd",
    importpath = "github.com/coreos/etcd",
    sum = "h1:f/Z3EoDSx1yjaIjLQGo1diYUlQYSBrrAQ5vP8NjwXwo=",
    version = "v3.3.17+incompatible",
)

go_repository(
    name = "com_github_coreos_go_etcd",
    importpath = "github.com/coreos/go-etcd",
    sum = "h1:bXhRBIXoTm9BYHS3gE0TtQuyNZyeEMux2sDi4oo5YOo=",
    version = "v2.0.0+incompatible",
)

go_repository(
    name = "com_github_coreos_go_oidc",
    importpath = "github.com/coreos/go-oidc",
    sum = "h1:sdJrfw8akMnCuUlaZU3tE/uYXFgfqom8DBE9so9EBsM=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_coreos_go_semver",
    importpath = "github.com/coreos/go-semver",
    sum = "h1:wkHLiw0WNATZnSG7epLsujiMCgPAc9xhjJ4tgnAxmfM=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_coreos_go_systemd",
    importpath = "github.com/coreos/go-systemd",
    sum = "h1:Wf6HqHfScWJN9/ZjdUKyjop4mf3Qdd+1TvvltAvM3m8=",
    version = "v0.0.0-20190321100706-95778dfbb74e",
)

go_repository(
    name = "com_github_coreos_pkg",
    importpath = "github.com/coreos/pkg",
    sum = "h1:lBNOc5arjvs8E5mO2tbpBpLoyyu8B6e44T7hJy6potg=",
    version = "v0.0.0-20180928190104-399ea9e2e55f",
)

go_repository(
    name = "com_github_cpu_goacmedns",
    importpath = "github.com/cpu/goacmedns",
    sum = "h1:QOeMpIEsIdm1LSASSswjaTf8CXmzcrgy5OeCfHjppA4=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_cpuguy83_go_md2man",
    importpath = "github.com/cpuguy83/go-md2man",
    sum = "h1:BSKMNlYxDvnunlTymqtgONjNnaRV1sTpcovwwjF22jk=",
    version = "v1.0.10",
)

go_repository(
    name = "com_github_cpuguy83_go_md2man_v2",
    importpath = "github.com/cpuguy83/go-md2man/v2",
    sum = "h1:EoUDS0afbrsXAZ9YQ9jdu/mZ2sXgT1/2yyNng4PGlyM=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_creack_pty",
    importpath = "github.com/creack/pty",
    sum = "h1:uDmaGzcdjhF4i/plgjmEsriH11Y0o7RKapEf/LDaM3w=",
    version = "v1.1.9",
)

go_repository(
    name = "com_github_cyphar_filepath_securejoin",
    importpath = "github.com/cyphar/filepath-securejoin",
    sum = "h1:jCwT2GTP+PY5nBz3c/YL5PAIbusElVrPujOBSCj8xRg=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_datadog_zstd",
    importpath = "github.com/DataDog/zstd",
    sum = "h1:3oxKN3wbHibqx897utPC2LTQU4J+IHWWJO+glkAkpFM=",
    version = "v1.4.1",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_daviddengcn_go_colortext",
    importpath = "github.com/daviddengcn/go-colortext",
    sum = "h1:uVsMphB1eRx7xB1njzL3fuMdWRN8HtVzoUOItHMwv5c=",
    version = "v0.0.0-20160507010035-511bcaf42ccd",
)

go_repository(
    name = "com_github_deislabs_oras",
    importpath = "github.com/deislabs/oras",
    sum = "h1:If674KraJVpujYR00rzdi0QAmW4BxzMJPVAZJKuhQ0c=",
    version = "v0.8.1",
)

go_repository(
    name = "com_github_denisenkom_go_mssqldb",
    importpath = "github.com/denisenkom/go-mssqldb",
    sum = "h1:83Wprp6ROGeiHFAP8WJdI2RoxALQYgdllERc3N5N2DM=",
    version = "v0.0.0-20191124224453-732737034ffd",
)

go_repository(
    name = "com_github_denverdino_aliyungo",
    importpath = "github.com/denverdino/aliyungo",
    sum = "h1:p6poVbjHDkKa+wtC8frBMwQtT3BmqGYBjzMwJ63tuR4=",
    version = "v0.0.0-20190125010748-a747050bb1ba",
)

go_repository(
    name = "com_github_devigned_tab",
    importpath = "github.com/devigned/tab",
    sum = "h1:3mD6Kb1mUOYeLpJvTVSDwSg5ZsfSxfvxGRTxRsJsITA=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_dgrijalva_jwt_go",
    importpath = "github.com/dgrijalva/jwt-go",
    sum = "h1:7qlOGliEKZXTDg6OTjfoBKDXWrumCAMpl/TFQ4/5kLM=",
    version = "v3.2.0+incompatible",
)

go_repository(
    name = "com_github_dgrijalva_jwt_go_v4",
    importpath = "github.com/dgrijalva/jwt-go/v4",
    sum = "h1:CaO/zOnF8VvUfEbhRatPcwKVWamvbYd8tQGRWacE9kU=",
    version = "v4.0.0-preview1",
)

go_repository(
    name = "com_github_dgryski_go_gk",
    importpath = "github.com/dgryski/go-gk",
    sum = "h1:XOPLOMn/zT4jIgxfxSsoXPxkrzz0FaCHwp33x5POJ+Q=",
    version = "v0.0.0-20200319235926-a69029f61654",
)

go_repository(
    name = "com_github_dgryski_go_sip13",
    importpath = "github.com/dgryski/go-sip13",
    sum = "h1:RMLoZVzv4GliuWafOuPuQDKSm1SJph7uCRnnS61JAn4=",
    version = "v0.0.0-20181026042036-e10d5fee7954",
)

go_repository(
    name = "com_github_digitalocean_godo",
    importpath = "github.com/digitalocean/godo",
    sum = "h1:IMElzMUpO1dVR8qjSg53+5vDkOLzMbhJt4yTAq7NGCQ=",
    version = "v1.44.0",
)

go_repository(
    name = "com_github_dimchansky_utfbom",
    importpath = "github.com/dimchansky/utfbom",
    sum = "h1:FcM3g+nofKgUteL8dm/UpdRXNC9KmADgTpLKsu0TRo4=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_djarvur_go_err113",
    importpath = "github.com/Djarvur/go-err113",
    sum = "h1:uCRZZOdMQ0TZPHYTdYpoC0bLYJKPEHPUJ8MeAa51lNU=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_djherbis_atime",
    importpath = "github.com/djherbis/atime",
    sum = "h1:ySLvBAM0EvOGaX7TI4dAM5lWj+RdJUCKtGSEHN8SGBg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_dnaeon_go_vcr",
    importpath = "github.com/dnaeon/go-vcr",
    sum = "h1:r8L/HqC0Hje5AXMu1ooW8oyQyOFv4GxqpL0nRP7SLLY=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_docker_cli",
    importpath = "github.com/docker/cli",
    sum = "h1:AbI1uj9w4yt6TvfKHfRu7G55KuQe7NCvWPQRKDoXggE=",
    version = "v0.0.0-20200210162036-a4bedce16568",
)

go_repository(
    name = "com_github_docker_distribution",
    importpath = "github.com/docker/distribution",
    sum = "h1:a5mlkVzth6W5A4fOsS3D2EO5BUmsJpcB+cRlLU7cSug=",
    version = "v2.7.1+incompatible",
)

go_repository(
    name = "com_github_docker_docker",
    importpath = "github.com/docker/docker",
    sum = "h1:IkZjBSIc8hBjLpqeAbeE5mca5mNgeatLHBy3GO78BWo=",
    version = "v1.13.1",
)

go_repository(
    name = "com_github_docker_docker_credential_helpers",
    importpath = "github.com/docker/docker-credential-helpers",
    sum = "h1:zI2p9+1NQYdnG6sMU26EX4aVGlqbInSQxQXLvzJ4RPQ=",
    version = "v0.6.3",
)

go_repository(
    name = "com_github_docker_go_connections",
    importpath = "github.com/docker/go-connections",
    sum = "h1:El9xVISelRB7BuFusrZozjnkIM5YnzCViNKohAFqRJQ=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_docker_go_metrics",
    importpath = "github.com/docker/go-metrics",
    sum = "h1:yWHOI+vFjEsAakUTSrtqc/SAHrhSkmn48pqjidZX3QA=",
    version = "v0.0.0-20180209012529-399ea8c73916",
)

go_repository(
    name = "com_github_docker_go_units",
    importpath = "github.com/docker/go-units",
    sum = "h1:3uh0PgVws3nIA0Q+MwDC8yjEPf9zjRfZZWXZYDct3Tw=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_docker_libtrust",
    importpath = "github.com/docker/libtrust",
    sum = "h1:ZClxb8laGDf5arXfYcAtECDFgAgHklGI8CxgjHnXKJ4=",
    version = "v0.0.0-20150114040149-fa567046d9b1",
)

go_repository(
    name = "com_github_docker_spdystream",
    importpath = "github.com/docker/spdystream",
    sum = "h1:cenwrSVm+Z7QLSV/BsnenAOcDXdX4cMv4wP0B/5QbPg=",
    version = "v0.0.0-20160310174837-449fdfce4d96",
)

go_repository(
    name = "com_github_docopt_docopt_go",
    importpath = "github.com/docopt/docopt-go",
    sum = "h1:bWDMxwH3px2JBh6AyO7hdCn/PkvCZXii8TGj7sbtEbQ=",
    version = "v0.0.0-20180111231733-ee0de3bc6815",
)

go_repository(
    name = "com_github_dsnet_compress",
    importpath = "github.com/dsnet/compress",
    sum = "h1:PlZu0n3Tuv04TzpfPbrnI0HW/YwodEXDS+oPKahKF0Q=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_dsnet_golib",
    importpath = "github.com/dsnet/golib",
    sum = "h1:tFh1tRc4CA31yP6qDcu+Trax5wW5GuMxvkIba07qVLY=",
    version = "v0.0.0-20171103203638-1ea166775780",
)

go_repository(
    name = "com_github_dustin_go_humanize",
    importpath = "github.com/dustin/go-humanize",
    sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_eapache_go_resiliency",
    importpath = "github.com/eapache/go-resiliency",
    sum = "h1:v7g92e/KSN71Rq7vSThKaWIq68fL4YHvWyiUKorFR1Q=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_eapache_go_xerial_snappy",
    importpath = "github.com/eapache/go-xerial-snappy",
    sum = "h1:YEetp8/yCZMuEPMUDHG0CW/brkkEp8mzqk2+ODEitlw=",
    version = "v0.0.0-20180814174437-776d5712da21",
)

go_repository(
    name = "com_github_eapache_queue",
    importpath = "github.com/eapache/queue",
    sum = "h1:YOEu7KNc61ntiQlcEeUIoDTJ2o8mQznoNvUhiigpIqc=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_elazarl_goproxy",
    importpath = "github.com/elazarl/goproxy",
    sum = "h1:yUdfgN0XgIJw7foRItutHYUIhlcKzcSf5vDpdhQAKTc=",
    version = "v0.0.0-20180725130230-947c36da3153",
)

go_repository(
    name = "com_github_emicklei_go_restful",
    importpath = "github.com/emicklei/go-restful",
    sum = "h1:spTtZBk5DYEvbxMVutUuTyh1Ao2r4iyvLdACqsl/Ljk=",
    version = "v2.9.5+incompatible",
)

go_repository(
    name = "com_github_emirpasic_gods",
    importpath = "github.com/emirpasic/gods",
    sum = "h1:QAUIPSaCu4G+POclxeqb3F+WPpdKqFGlw36+yOzGlrg=",
    version = "v1.12.0",
)

go_repository(
    name = "com_github_envoyproxy_go_control_plane",
    importpath = "github.com/envoyproxy/go-control-plane",
    sum = "h1:rEvIZUSZ3fx39WIi3JkQqQBitGwpELBIYWeBVh6wn+E=",
    version = "v0.9.4",
)

go_repository(
    name = "com_github_envoyproxy_protoc_gen_validate",
    importpath = "github.com/envoyproxy/protoc-gen-validate",
    sum = "h1:EQciDnbrYxy13PgWoY8AqoxGiPrpgBZ1R8UNe3ddc+A=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_erikstmartin_go_testdb",
    importpath = "github.com/erikstmartin/go-testdb",
    sum = "h1:Yzb9+7DPaBjB8zlTR87/ElzFsnQfuHnVUVqpZZIcV5Y=",
    version = "v0.0.0-20160219214506-8d10e4a1bae5",
)

go_repository(
    name = "com_github_evanphx_json_patch",
    importpath = "github.com/evanphx/json-patch",
    sum = "h1:kLcOMZeuLAJvL2BPWLMIj5oaZQobrkAqrL+WFZwQses=",
    version = "v4.9.0+incompatible",
)

go_repository(
    name = "com_github_exponent_io_jsonpath",
    importpath = "github.com/exponent-io/jsonpath",
    sum = "h1:105gxyaGwCFad8crR9dcMQWvV9Hvulu6hwUh4tWPJnM=",
    version = "v0.0.0-20151013193312-d6023ce2651d",
)

go_repository(
    name = "com_github_fatih_camelcase",
    importpath = "github.com/fatih/camelcase",
    sum = "h1:hxNvNX/xYBp0ovncs8WyWZrOrpBNub/JfaMvbURyft8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_fatih_color",
    importpath = "github.com/fatih/color",
    sum = "h1:8xPHl4/q1VyqGIPif1F+1V3Y3lSmrq01EabUW3CoW5s=",
    version = "v1.9.0",
)

go_repository(
    name = "com_github_fatih_structs",
    importpath = "github.com/fatih/structs",
    sum = "h1:Q7juDM0QtcnhCpeyLGQKyg4TOIghuNXrkL32pHAUMxo=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_flynn_go_shlex",
    importpath = "github.com/flynn/go-shlex",
    sum = "h1:BHsljHzVlRcyQhjrss6TZTdY2VfCqZPbv5k3iBFa2ZQ=",
    version = "v0.0.0-20150515145356-3f9db97f8568",
)

go_repository(
    name = "com_github_fortytw2_leaktest",
    importpath = "github.com/fortytw2/leaktest",
    sum = "h1:u8491cBMTQ8ft8aeV+adlcytMZylmA5nnwwkRZjI8vw=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_frankban_quicktest",
    importpath = "github.com/frankban/quicktest",
    sum = "h1:PvpJR0Uq8SdX+zagCMsarBMlhz6ysGTf1+pRmCsRXqY=",
    version = "v1.8.1",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    importpath = "github.com/fsnotify/fsnotify",
    sum = "h1:hsms1Qyu0jgnwNXIxa+/V/PDsU6CfLf6CNO8H7IWoS4=",
    version = "v1.4.9",
)

go_repository(
    name = "com_github_fsouza_fake_gcs_server",
    importpath = "github.com/fsouza/fake-gcs-server",
    sum = "h1:3bRRh/rQnB2XbrMolHAj9oX/PFiWVQFVVfPR5y2pxb8=",
    version = "v1.19.4",
)

go_repository(
    name = "com_github_fvbommel_sortorder",
    importpath = "github.com/fvbommel/sortorder",
    sum = "h1:dSnXLt4mJYH25uDDGa3biZNQsozaUWDSWeKJ0qqFfzE=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_garyburd_redigo",
    importpath = "github.com/garyburd/redigo",
    sum = "h1:LofdAjjjqCSXMwLGgOgnE+rdPuvX9DxCqaHwKy7i/ko=",
    version = "v0.0.0-20150301180006-535138d7bcd7",
)

go_repository(
    name = "com_github_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_gliderlabs_ssh",
    importpath = "github.com/gliderlabs/ssh",
    sum = "h1:6zsha5zo/TWhRhwqCD3+EarCAgZ2yN28ipRnGPnwkI0=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_globalsign_mgo",
    importpath = "github.com/globalsign/mgo",
    sum = "h1:DujepqpGd1hyOd7aW59XpK7Qymp8iy83xq74fLr21is=",
    version = "v0.0.0-20181015135952-eeefdecb41b8",
)

go_repository(
    name = "com_github_go_bindata_go_bindata_v3",
    importpath = "github.com/go-bindata/go-bindata/v3",
    sum = "h1:F0nVttLC3ws0ojc7p60veTurcOm//D4QBODNM7EGrCI=",
    version = "v3.1.3",
)

go_repository(
    name = "com_github_go_critic_go_critic",
    importpath = "github.com/go-critic/go-critic",
    sum = "h1:sGEEdiuvLV0OC7/yC6MnK3K6LCPBplspK45B0XVdFAc=",
    version = "v0.4.3",
)

go_repository(
    name = "com_github_go_gl_glfw",
    importpath = "github.com/go-gl/glfw",
    sum = "h1:QbL/5oDUmRBzO9/Z7Seo6zf912W/a6Sr4Eu0G/3Jho0=",
    version = "v0.0.0-20190409004039-e6da0acd62b1",
)

go_repository(
    name = "com_github_go_gl_glfw_v3_3_glfw",
    importpath = "github.com/go-gl/glfw/v3.3/glfw",
    sum = "h1:WtGNWLvXpe6ZudgnXrq0barxBImvnnJoMEhXAzcbM0I=",
    version = "v0.0.0-20200222043503-6f7a984d4dc4",
)

go_repository(
    name = "com_github_go_ini_ini",
    importpath = "github.com/go-ini/ini",
    sum = "h1:0wVcG9udk2C3TGgmdIGKK9ScOZHZB5nbG+gwji9fhhc=",
    version = "v1.55.0",
)

go_repository(
    name = "com_github_go_kit_kit",
    importpath = "github.com/go-kit/kit",
    sum = "h1:wDJmvq38kDhkVxi50ni9ykkdUr1PKgqKOoi01fa0Mdk=",
    version = "v0.9.0",
)

go_repository(
    name = "com_github_go_ldap_ldap",
    importpath = "github.com/go-ldap/ldap",
    sum = "h1:kD5HQcAzlQ7yrhfn+h+MSABeAy/jAJhvIJ/QDllP44g=",
    version = "v3.0.2+incompatible",
)

go_repository(
    name = "com_github_go_lintpack_lintpack",
    importpath = "github.com/go-lintpack/lintpack",
    sum = "h1:DI5mA3+eKdWeJ40nU4d6Wc26qmdG8RCi/btYq0TuRN0=",
    version = "v0.5.2",
)

go_repository(
    name = "com_github_go_logfmt_logfmt",
    importpath = "github.com/go-logfmt/logfmt",
    sum = "h1:MP4Eh7ZCb31lleYCFuwm0oe4/YGak+5l1vA2NOE80nA=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_go_logr_logr",
    importpath = "github.com/go-logr/logr",
    sum = "h1:fV3MLmabKIZ383XifUjFSwcoGee0v9qgPp8wy5svibE=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_go_logr_zapr",
    importpath = "github.com/go-logr/zapr",
    sum = "h1:v6Ji8yBW77pva6NkJKQdHLAJKrIJKRHz0RXwPqCHSR4=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_go_ole_go_ole",
    importpath = "github.com/go-ole/go-ole",
    sum = "h1:2lOsA72HgjxAuMlKpFiCbHTvu44PIVkZ5hqm3RSdI/E=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_go_openapi_analysis",
    importpath = "github.com/go-openapi/analysis",
    sum = "h1:8b2ZgKfKIUTVQpTb77MoRDIMEIwvDVw40o3aOXdfYzI=",
    version = "v0.19.5",
)

go_repository(
    name = "com_github_go_openapi_errors",
    importpath = "github.com/go-openapi/errors",
    sum = "h1:a2kIyV3w+OS3S97zxUndRVD46+FhGOUBDFY7nmu4CsY=",
    version = "v0.19.2",
)

go_repository(
    name = "com_github_go_openapi_jsonpointer",
    importpath = "github.com/go-openapi/jsonpointer",
    sum = "h1:gihV7YNZK1iK6Tgwwsxo2rJbD1GTbdm72325Bq8FI3w=",
    version = "v0.19.3",
)

go_repository(
    name = "com_github_go_openapi_jsonreference",
    importpath = "github.com/go-openapi/jsonreference",
    sum = "h1:5cxNfTy0UVC3X8JL5ymxzyoUZmo8iZb+jeTWn7tUa8o=",
    version = "v0.19.3",
)

go_repository(
    name = "com_github_go_openapi_loads",
    importpath = "github.com/go-openapi/loads",
    sum = "h1:5I4CCSqoWzT+82bBkNIvmLc0UOsoKKQ4Fz+3VxOB7SY=",
    version = "v0.19.4",
)

go_repository(
    name = "com_github_go_openapi_runtime",
    importpath = "github.com/go-openapi/runtime",
    sum = "h1:csnOgcgAiuGoM/Po7PEpKDoNulCcF3FGbSnbHfxgjMI=",
    version = "v0.19.4",
)

go_repository(
    name = "com_github_go_openapi_spec",
    importpath = "github.com/go-openapi/spec",
    sum = "h1:rMMMj8cV38KVXK7SFc+I2MWClbEfbK705+j+dyqun5g=",
    version = "v0.19.6",
)

go_repository(
    name = "com_github_go_openapi_strfmt",
    importpath = "github.com/go-openapi/strfmt",
    sum = "h1:eRfyY5SkaNJCAwmmMcADjY31ow9+N7MCLW7oRkbsINA=",
    version = "v0.19.3",
)

go_repository(
    name = "com_github_go_openapi_swag",
    importpath = "github.com/go-openapi/swag",
    sum = "h1:VRuXN2EnMSsZdauzdss6JBC29YotDqG59BZ+tdlIL1s=",
    version = "v0.19.7",
)

go_repository(
    name = "com_github_go_openapi_validate",
    importpath = "github.com/go-openapi/validate",
    sum = "h1:QhCBKRYqZR+SKo4gl1lPhPahope8/RLt6EVgY8X80w0=",
    version = "v0.19.5",
)

go_repository(
    name = "com_github_go_sql_driver_mysql",
    importpath = "github.com/go-sql-driver/mysql",
    sum = "h1:ozyZYNQW3x3HtqT1jira07DN2PArx2v7/mN66gGcHOs=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_go_stack_stack",
    importpath = "github.com/go-stack/stack",
    sum = "h1:5SgMzNM5HxrEjV0ww2lTmX6E2Izsfxas4+YHWRs3Lsk=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_go_test_deep",
    importpath = "github.com/go-test/deep",
    sum = "h1:u2CU3YKy9I2pmu9pX0eq50wCgjfGIt539SqR7FbHiho=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_go_toolsmith_astcast",
    importpath = "github.com/go-toolsmith/astcast",
    sum = "h1:JojxlmI6STnFVG9yOImLeGREv8W2ocNUM+iOhR6jE7g=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_astcopy",
    importpath = "github.com/go-toolsmith/astcopy",
    sum = "h1:OMgl1b1MEpjFQ1m5ztEO06rz5CUd3oBv9RF7+DyvdG8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_astequal",
    importpath = "github.com/go-toolsmith/astequal",
    sum = "h1:4zxD8j3JRFNyLN46lodQuqz3xdKSrur7U/sr0SDS/gQ=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_astfmt",
    importpath = "github.com/go-toolsmith/astfmt",
    sum = "h1:A0vDDXt+vsvLEdbMFJAUBI/uTbRw1ffOPnxsILnFL6k=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_astinfo",
    importpath = "github.com/go-toolsmith/astinfo",
    sum = "h1:wP6mXeB2V/d1P1K7bZ5vDUO3YqEzcvOREOxZPEu3gVI=",
    version = "v0.0.0-20180906194353-9809ff7efb21",
)

go_repository(
    name = "com_github_go_toolsmith_astp",
    importpath = "github.com/go-toolsmith/astp",
    sum = "h1:alXE75TXgcmupDsMK1fRAy0YUzLzqPVvBKoyWV+KPXg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_pkgload",
    importpath = "github.com/go-toolsmith/pkgload",
    sum = "h1:4DFWWMXVfbcN5So1sBNW9+yeiMqLFGl1wFLTL5R0Tgg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_strparse",
    importpath = "github.com/go-toolsmith/strparse",
    sum = "h1:Vcw78DnpCAKlM20kSbAyO4mPfJn/lyYA4BJUDxe2Jb4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_toolsmith_typep",
    importpath = "github.com/go-toolsmith/typep",
    sum = "h1:8xdsa1+FSIH/RhEkgnD1j2CJOy5mNllW1Q9tRiYwvlk=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_go_xmlfmt_xmlfmt",
    importpath = "github.com/go-xmlfmt/xmlfmt",
    sum = "h1:khEcpUM4yFcxg4/FHQWkvVRmgijNXRfzkIDHh23ggEo=",
    version = "v0.0.0-20191208150333-d5b6f63a941b",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    importpath = "github.com/go-yaml/yaml",
    sum = "h1:RYi2hDdss1u4YE7GwixGzWwVo47T8UQwnTLB6vQiq+o=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_gobuffalo_envy",
    importpath = "github.com/gobuffalo/envy",
    sum = "h1:OQl5ys5MBea7OGCdvPbBJWRgnhC/fGona6QKfvFeau8=",
    version = "v1.7.1",
)

go_repository(
    name = "com_github_gobuffalo_flect",
    importpath = "github.com/gobuffalo/flect",
    sum = "h1:EWCvMGGxOjsgwlWaP+f4+Hh6yrrte7JeFL2S6b+0hdM=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_gobwas_glob",
    importpath = "github.com/gobwas/glob",
    sum = "h1:A4xDbljILXROh+kObIiy5kIaPYD8e96x1tgBhUI5J+Y=",
    version = "v0.2.3",
)

go_repository(
    name = "com_github_godbus_dbus",
    importpath = "github.com/godbus/dbus",
    sum = "h1:BWhy2j3IXJhjCbC68FptL43tDKIq8FladmaTs3Xs7Z8=",
    version = "v0.0.0-20190422162347-ade71ed3457e",
)

go_repository(
    name = "com_github_gofrs_flock",
    importpath = "github.com/gofrs/flock",
    sum = "h1:DP+LD/t0njgoPBvT5MJLeliUIVQR03hiKR6vezdwHlc=",
    version = "v0.7.1",
)

go_repository(
    name = "com_github_gogo_googleapis",
    importpath = "github.com/gogo/googleapis",
    sum = "h1:kFkMAZBNAn4j7K0GiZr8cRYzejq68VbheufiV3YuyFI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_gogo_protobuf",
    importpath = "github.com/gogo/protobuf",
    sum = "h1:DqDEcV5aeaTmdFBePNpYsp3FlcVH/2ISVVM9Qf8PSls=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_golang_gddo",
    importpath = "github.com/golang/gddo",
    sum = "h1:KRMr9A3qfbVM7iV/WcLY/rL5LICqwMHLhwRXKu99fXw=",
    version = "v0.0.0-20190419222130-af0f2af80721",
)

go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repository(
    name = "com_github_golang_groupcache",
    importpath = "github.com/golang/groupcache",
    sum = "h1:1r7pUrabqp18hOBcwBwiTsbnFeTZHV9eER/QT5JVZxY=",
    version = "v0.0.0-20200121045136-8c9f03a8e57e",
)

go_repository(
    name = "com_github_golang_lint",
    importpath = "github.com/golang/lint",
    sum = "h1:2hRPrmiwPrp3fQX967rNJIhQPtiGXdlQWAxKbKw3VHA=",
    version = "v0.0.0-20180702182130-06c8688daad7",
)

go_repository(
    name = "com_github_golang_mock",
    importpath = "github.com/golang/mock",
    sum = "h1:jlYHihg//f7RRwuPfptm04yp4s7O6Kw8EZiVYIGcH0g=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_golang_protobuf",
    importpath = "github.com/golang/protobuf",
    sum = "h1:+Z5KGCizgyZCbGh1KZqA0fcLLkwbsjIzS4aV2v7wJX0=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_golang_snappy",
    importpath = "github.com/golang/snappy",
    sum = "h1:Qgr9rKW7uDUkrbSmQeiDsGa8SjGyCOGtuasMWwvp2P4=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_golang_sql_civil",
    importpath = "github.com/golang-sql/civil",
    sum = "h1:lXe2qZdvpiX5WZkZR4hgp4KJVfY3nMkvmwbVkpv1rVY=",
    version = "v0.0.0-20190719163853-cb61b32ac6fe",
)

go_repository(
    name = "com_github_golangci_check",
    importpath = "github.com/golangci/check",
    sum = "h1:23T5iq8rbUYlhpt5DB4XJkc6BU31uODLD1o1gKvZmD0=",
    version = "v0.0.0-20180506172741-cfe4005ccda2",
)

go_repository(
    name = "com_github_golangci_dupl",
    importpath = "github.com/golangci/dupl",
    sum = "h1:w8hkcTqaFpzKqonE9uMCefW1WDie15eSP/4MssdenaM=",
    version = "v0.0.0-20180902072040-3e9179ac440a",
)

go_repository(
    name = "com_github_golangci_errcheck",
    importpath = "github.com/golangci/errcheck",
    sum = "h1:YYWNAGTKWhKpcLLt7aSj/odlKrSrelQwlovBpDuf19w=",
    version = "v0.0.0-20181223084120-ef45e06d44b6",
)

go_repository(
    name = "com_github_golangci_go_misc",
    importpath = "github.com/golangci/go-misc",
    sum = "h1:9kfjN3AdxcbsZBf8NjltjWihK2QfBBBZuv91cMFfDHw=",
    version = "v0.0.0-20180628070357-927a3d87b613",
)

go_repository(
    name = "com_github_golangci_goconst",
    importpath = "github.com/golangci/goconst",
    sum = "h1:pe9JHs3cHHDQgOFXJJdYkK6fLz2PWyYtP4hthoCMvs8=",
    version = "v0.0.0-20180610141641-041c5f2b40f3",
)

go_repository(
    name = "com_github_golangci_gocyclo",
    importpath = "github.com/golangci/gocyclo",
    sum = "h1:pXTK/gkVNs7Zyy7WKgLXmpQ5bHTrq5GDsp8R9Qs67g0=",
    version = "v0.0.0-20180528144436-0a533e8fa43d",
)

go_repository(
    name = "com_github_golangci_gofmt",
    importpath = "github.com/golangci/gofmt",
    sum = "h1:iR3fYXUjHCR97qWS8ch1y9zPNsgXThGwjKPrYfqMPks=",
    version = "v0.0.0-20190930125516-244bba706f1a",
)

go_repository(
    name = "com_github_golangci_golangci_lint",
    importpath = "github.com/golangci/golangci-lint",
    sum = "h1:VYLx63qb+XJsHdZ27PMS2w5JZacN0XG8ffUwe7yQomo=",
    version = "v1.27.0",
)

go_repository(
    name = "com_github_golangci_ineffassign",
    importpath = "github.com/golangci/ineffassign",
    sum = "h1:gLLhTLMk2/SutryVJ6D4VZCU3CUqr8YloG7FPIBWFpI=",
    version = "v0.0.0-20190609212857-42439a7714cc",
)

go_repository(
    name = "com_github_golangci_lint_1",
    importpath = "github.com/golangci/lint-1",
    sum = "h1:MfyDlzVjl1hoaPzPD4Gpb/QgoRfSBR0jdhwGyAWwMSA=",
    version = "v0.0.0-20191013205115-297bf364a8e0",
)

go_repository(
    name = "com_github_golangci_maligned",
    importpath = "github.com/golangci/maligned",
    sum = "h1:kNY3/svz5T29MYHubXix4aDDuE3RWHkPvopM/EDv/MA=",
    version = "v0.0.0-20180506175553-b1d89398deca",
)

go_repository(
    name = "com_github_golangci_misspell",
    importpath = "github.com/golangci/misspell",
    sum = "h1:pLzmVdl3VxTOncgzHcvLOKirdvcx/TydsClUQXTehjo=",
    version = "v0.3.5",
)

go_repository(
    name = "com_github_golangci_prealloc",
    importpath = "github.com/golangci/prealloc",
    sum = "h1:leSNB7iYzLYSSx3J/s5sVf4Drkc68W2wm4Ixh/mr0us=",
    version = "v0.0.0-20180630174525-215b22d4de21",
)

go_repository(
    name = "com_github_golangci_revgrep",
    importpath = "github.com/golangci/revgrep",
    sum = "h1:XQKc8IYQOeRwVs36tDrEmTgDgP88d5iEURwpmtiAlOM=",
    version = "v0.0.0-20180812185044-276a5c0a1039",
)

go_repository(
    name = "com_github_golangci_unconvert",
    importpath = "github.com/golangci/unconvert",
    sum = "h1:zwtduBRr5SSWhqsYNgcuWO2kFlpdOZbP0+yRjmvPGys=",
    version = "v0.0.0-20180507085042-28b1c447d1f4",
)

go_repository(
    name = "com_github_golangplus_bytes",
    importpath = "github.com/golangplus/bytes",
    sum = "h1:7xqw01UYS+KCI25bMrPxwNYkSns2Db1ziQPpVq99FpE=",
    version = "v0.0.0-20160111154220-45c989fe5450",
)

go_repository(
    name = "com_github_golangplus_fmt",
    importpath = "github.com/golangplus/fmt",
    sum = "h1:f5gsjBiF9tRRVomCvrkGMMWI8W1f2OBFar2c5oakAP0=",
    version = "v0.0.0-20150411045040-2a5d6d7d2995",
)

go_repository(
    name = "com_github_golangplus_testing",
    importpath = "github.com/golangplus/testing",
    sum = "h1:KhcknUwkWHKZPbFy2P7jH5LKJ3La+0ZeknkkmrSgqb0=",
    version = "v0.0.0-20180327235837-af21d9c3145e",
)

go_repository(
    name = "com_github_gomodule_redigo",
    importpath = "github.com/gomodule/redigo",
    sum = "h1:ZKld1VOtsGhAe37E7wMxEDgAlGM5dvFY+DiOhSkhP9Y=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_google_btree",
    importpath = "github.com/google/btree",
    sum = "h1:0udJVsspx3VBr5FwtLhQQtuAsVc79tTq0ocGIPAU6qo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_google_go_cmp",
    importpath = "github.com/google/go-cmp",
    sum = "h1:L8R9j+yAqZuZjsqh/z+F1NCffTKKLShY6zXTItVIZ8M=",
    version = "v0.5.4",
)

go_repository(
    name = "com_github_google_go_containerregistry",
    importpath = "github.com/google/go-containerregistry",
    sum = "h1:AG8FSAfXglim2l5qSrqp5VK2Xl03PiBf25NiTGGamws=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_google_go_github",
    importpath = "github.com/google/go-github",
    sum = "h1:N0LgJ1j65A7kfXrZnUDaYCs/Sf4rEjNlfyDHW9dolSY=",
    version = "v17.0.0+incompatible",
)

go_repository(
    name = "com_github_google_go_github_v27",
    importpath = "github.com/google/go-github/v27",
    sum = "h1:oiOZuBmGHvrGM1X9uNUAUlLgp5r1UUO/M/KnbHnLRlQ=",
    version = "v27.0.6",
)

go_repository(
    name = "com_github_google_go_github_v28",
    importpath = "github.com/google/go-github/v28",
    sum = "h1:kORf5ekX5qwXO2mGzXXOjMe/g6ap8ahVe0sBEulhSxo=",
    version = "v28.1.1",
)

go_repository(
    name = "com_github_google_go_github_v29",
    importpath = "github.com/google/go-github/v29",
    sum = "h1:IktKCTwU//aFHnpA+2SLIi7Oo9uhAzgsdZNbcAqhgdc=",
    version = "v29.0.3",
)

go_repository(
    name = "com_github_google_go_github_v32",
    importpath = "github.com/google/go-github/v32",
    sum = "h1:q74KVb22spUq0U5HqZ9VCYqQz8YRuOtL/39ZnfwO+NM=",
    version = "v32.0.0",
)

go_repository(
    name = "com_github_google_go_licenses",
    importpath = "github.com/google/go-licenses",
    sum = "h1:eVR9gT5gBPTHXeyGAcA8OF/SKNUFFg+a0BJqfx4z5eE=",
    version = "v0.0.0-20200227160636-0fa8c766a591",
)

go_repository(
    name = "com_github_google_go_querystring",
    importpath = "github.com/google/go-querystring",
    sum = "h1:Xkwi/a1rcvNg1PPYe5vI8GbeBY/jrVuDX5ASuANWTrk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_google_go_replayers_grpcreplay",
    importpath = "github.com/google/go-replayers/grpcreplay",
    sum = "h1:eNb1y9rZFmY4ax45uEEECSa8fsxGRU+8Bil52ASAwic=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_google_go_replayers_httpreplay",
    importpath = "github.com/google/go-replayers/httpreplay",
    sum = "h1:AX7FUb4BjrrzNvblr/OlgwrmFiep6soj5K2QSDW7BGk=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_google_gofuzz",
    importpath = "github.com/google/gofuzz",
    sum = "h1:xRy4A+RhZaiKjJ1bPfwQ8sedCA+YS2YcCHW6ec7JMi0=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_google_licenseclassifier",
    importpath = "github.com/google/licenseclassifier",
    sum = "h1:OggOMmdI0JLwg1FkOKH9S7fVHF0oEm8PX6S8kAdpOps=",
    version = "v0.0.0-20200402202327-879cb1424de0",
)

go_repository(
    name = "com_github_google_mako",
    importpath = "github.com/google/mako",
    sum = "h1:/o5e44nTD/QEEiWPGSFT3bSqcq3Qg7q27N9bv4gKh5M=",
    version = "v0.0.0-20190821191249-122f8dcef9e3",
)

go_repository(
    name = "com_github_google_martian",
    importpath = "github.com/google/martian",
    sum = "h1:xmapqc1AyLoB+ddYT6r04bD9lIjlOqGaREovi0SzFaE=",
    version = "v2.1.1-0.20190517191504-25dcb96d9e51+incompatible",
)

go_repository(
    name = "com_github_google_martian_v3",
    importpath = "github.com/google/martian/v3",
    sum = "h1:pMen7vLs8nvgEYhywH3KDWJIJTeEr2ULsVWHWYHQyBs=",
    version = "v3.0.0",
)

go_repository(
    name = "com_github_google_pprof",
    importpath = "github.com/google/pprof",
    sum = "h1:k+KkMRk8mGOu1xG38StS7dQ+Z6oW1i9n3dgrAVU9Q/E=",
    version = "v0.0.0-20200905233945-acf8798be1f7",
)

go_repository(
    name = "com_github_google_renameio",
    importpath = "github.com/google/renameio",
    sum = "h1:GOZbcHa3HfsPKPlmyPyN2KEohoMXOhdMbHrvbpl2QaA=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_google_rpmpack",
    importpath = "github.com/google/rpmpack",
    sum = "h1:BW6OvS3kpT5UEPbCZ+KyX/OB4Ks9/MNMhWjqPPkZxsE=",
    version = "v0.0.0-20191226140753-aa36bfddb3a0",
)

go_repository(
    name = "com_github_google_subcommands",
    importpath = "github.com/google/subcommands",
    sum = "h1:/eqq+otEXm5vhfBrbREPCSVQbvofip6kIz+mX5TUH7k=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_google_uuid",
    importpath = "github.com/google/uuid",
    sum = "h1:Gkbcsh/GbpXz7lPftLA3P6TYMwjCLYm83jiFQZF/3gY=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_google_wire",
    importpath = "github.com/google/wire",
    sum = "h1:kXcsA/rIGzJImVqPdhfnr6q0xsS9gU0515q1EPpJ9fE=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_googleapis_gax_go",
    importpath = "github.com/googleapis/gax-go",
    sum = "h1:silFMLAnr330+NRuag/VjIGF7TLp/LBrV2CJKFLWEww=",
    version = "v2.0.2+incompatible",
)

go_repository(
    name = "com_github_googleapis_gax_go_v2",
    importpath = "github.com/googleapis/gax-go/v2",
    sum = "h1:sjZBwGj9Jlw33ImPtvFviGYvseOtDM7hkSKB7+Tv3SM=",
    version = "v2.0.5",
)

go_repository(
    name = "com_github_googleapis_gnostic",
    importpath = "github.com/googleapis/gnostic",
    sum = "h1:A8Yhf6EtqTv9RMsU6MQTyrtV1TjWlR6xU9BsZIwuTCM=",
    version = "v0.5.1",
)

go_repository(
    name = "com_github_googlecloudplatform_cloud_builders_gcs_fetcher",
    importpath = "github.com/GoogleCloudPlatform/cloud-builders/gcs-fetcher",
    sum = "h1:Pjo3SOZigEnIGevhFqcbFndnqyCH8WimcREd3hRM9vU=",
    version = "v0.0.0-20191203181535-308b93ad1f39",
)

go_repository(
    name = "com_github_googlecloudplatform_cloudsql_proxy",
    importpath = "github.com/GoogleCloudPlatform/cloudsql-proxy",
    sum = "h1:sTOp2Ajiew5XIH92YSdwhYc+bgpUX5j5TKK/Ac8Saw8=",
    version = "v0.0.0-20191009163259-e802c2cb94ae",
)

go_repository(
    name = "com_github_googlecloudplatform_k8s_cloud_provider",
    importpath = "github.com/GoogleCloudPlatform/k8s-cloud-provider",
    sum = "h1:N7lSsF+R7wSulUADi36SInSQA3RvfO/XclHQfedr0qk=",
    version = "v0.0.0-20190822182118-27a4ced34534",
)

go_repository(
    name = "com_github_googlecloudplatform_testgrid",
    importpath = "github.com/GoogleCloudPlatform/testgrid",
    sum = "h1:Os7plKvCWxu37kdaqaHC+pvCva1nvsRuhVvMSPyhEqA=",
    version = "v0.0.30",
)

go_repository(
    name = "com_github_gookit_color",
    importpath = "github.com/gookit/color",
    sum = "h1:xOYBan3Fwlrqj1M1UN2TlHOCRiek3bGzWf/vPnJ1roE=",
    version = "v1.2.4",
)

go_repository(
    name = "com_github_gophercloud_gophercloud",
    importpath = "github.com/gophercloud/gophercloud",
    sum = "h1:P/nh25+rzXouhytV2pUHBb65fnds26Ghl8/391+sT5o=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_gopherjs_gopherjs",
    importpath = "github.com/gopherjs/gopherjs",
    sum = "h1:EGx4pi6eqNxGaHF6qqu48+N2wcFQ5qg5FXgOdqsJ5d8=",
    version = "v0.0.0-20181017120253-0766667cb4d1",
)

go_repository(
    name = "com_github_goreleaser_goreleaser",
    importpath = "github.com/goreleaser/goreleaser",
    sum = "h1:Z+7XPrfGK11s/Sp+a06sx2FzGuCjTBdxN2ubpGvQbjY=",
    version = "v0.136.0",
)

go_repository(
    name = "com_github_goreleaser_nfpm",
    importpath = "github.com/goreleaser/nfpm",
    sum = "h1:BPwIomC+e+yuDX9poJowzV7JFVcYA0+LwGSkbAPs2Hw=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_gorilla_context",
    importpath = "github.com/gorilla/context",
    sum = "h1:AWwleXJkX/nhcU9bZSnZoi3h/qGYqQAGhq6zZe/aQW8=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_gorilla_csrf",
    importpath = "github.com/gorilla/csrf",
    sum = "h1:QqQ/OWwuFp4jMKgBFAzJVW3FMULdyUW7JoM4pEWuqKg=",
    version = "v1.6.2",
)

go_repository(
    name = "com_github_gorilla_handlers",
    importpath = "github.com/gorilla/handlers",
    sum = "h1:0QniY0USkHQ1RGCLfKxeNHK9bkDHGRYGNDFBCS+YARg=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    sum = "h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_gorilla_securecookie",
    importpath = "github.com/gorilla/securecookie",
    sum = "h1:miw7JPhV+b/lAHSXz4qd/nN9jRiAFV5FwjeKyCS8BvQ=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_gorilla_sessions",
    importpath = "github.com/gorilla/sessions",
    sum = "h1:S7P+1Hm5V/AT9cjEcUD5uDaQSX0OE577aCXgoaKpYbQ=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_gorilla_websocket",
    importpath = "github.com/gorilla/websocket",
    sum = "h1:+/TMaTYc4QFitKJxsQ7Yye35DkWvkdLcvGKqM+x0Ufc=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_gostaticanalysis_analysisutil",
    importpath = "github.com/gostaticanalysis/analysisutil",
    sum = "h1:iwp+5/UAyzQSFgQ4uR2sni99sJ8Eo9DEacKWM5pekIg=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_gosuri_uitable",
    importpath = "github.com/gosuri/uitable",
    sum = "h1:IG2xLKRvErL3uhY6e1BylFzG+aJiwQviDDTfOKeKTpY=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_gotestyourself_gotestyourself",
    importpath = "github.com/gotestyourself/gotestyourself",
    sum = "h1:AQwinXlbQR2HvPjQZOmDhRqsv5mZf+Jb1RnSLxcqZcI=",
    version = "v2.2.0+incompatible",
)

go_repository(
    name = "com_github_gregjones_httpcache",
    importpath = "github.com/gregjones/httpcache",
    sum = "h1:f8eY6cV/x1x+HLjOp4r72s/31/V2aTUtg5oKRRPf8/Q=",
    version = "v0.0.0-20190212212710-3befbb6ad0cc",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_middleware",
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
    sum = "h1:z53tR0945TRRQO/fLEVPI6SMv7ZflF0TEaTAoU7tOzg=",
    version = "v1.0.1-0.20190118093823-f849b5445de4",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
    sum = "h1:Ovs26xHkKqVztRpIrF/92BcuyuQ/YW4NSIpoGtfXNho=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_grpc_ecosystem_grpc_gateway",
    importpath = "github.com/grpc-ecosystem/grpc-gateway",
    sum = "h1:D0EVSTwQoQOyfY35QNSuPJA4jpZRtkoGYWQMB7XNg5o=",
    version = "v1.12.2",
)

go_repository(
    name = "com_github_h2non_gock",
    importpath = "github.com/h2non/gock",
    sum = "h1:17gCehSo8ZOgEsFKpQgqHiR7VLyjxdAG3lkhVvO9QZU=",
    version = "v1.0.9",
)

go_repository(
    name = "com_github_hashicorp_consul_api",
    importpath = "github.com/hashicorp/consul/api",
    sum = "h1:BNQPM9ytxj6jbjjdRPioQ94T6YXriSopn0i8COv6SRA=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_hashicorp_consul_sdk",
    importpath = "github.com/hashicorp/consul/sdk",
    sum = "h1:LnuDWGNsoajlhGyHJvuWW6FVqRl8JOTPqS6CPTsYjhY=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    importpath = "github.com/hashicorp/errwrap",
    sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_go_cleanhttp",
    importpath = "github.com/hashicorp/go-cleanhttp",
    sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
    version = "v0.5.1",
)

go_repository(
    name = "com_github_hashicorp_go_hclog",
    importpath = "github.com/hashicorp/go-hclog",
    sum = "h1:CG6TE5H9/JXsFWJCfoIVpKFIkFe6ysEuHirp4DxCsHI=",
    version = "v0.9.2",
)

go_repository(
    name = "com_github_hashicorp_go_immutable_radix",
    importpath = "github.com/hashicorp/go-immutable-radix",
    sum = "h1:AKDB1HM5PWEA7i4nhcpwOrO2byshxBjXVn/J/3+z5/0=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_go_msgpack",
    importpath = "github.com/hashicorp/go-msgpack",
    sum = "h1:zKjpN5BK/P5lMYrLmBHdBULWbJ0XpYR+7NGzqkZzoD4=",
    version = "v0.5.3",
)

go_repository(
    name = "com_github_hashicorp_go_multierror",
    importpath = "github.com/hashicorp/go-multierror",
    sum = "h1:B9UzwGQJehnUY1yNrnwREHc3fGbC2xefo8g4TbElacI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_hashicorp_go_net",
    importpath = "github.com/hashicorp/go.net",
    sum = "h1:sNCoNyDEvN1xa+X0baata4RdcpKwcMS6DH+xwfqPgjw=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_hashicorp_go_plugin",
    importpath = "github.com/hashicorp/go-plugin",
    sum = "h1:4OtAfUGbnKC6yS48p0CtMX2oFYtzFZVv6rok3cRWgnE=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_hashicorp_go_retryablehttp",
    importpath = "github.com/hashicorp/go-retryablehttp",
    sum = "h1:HJunrbHTDDbBb/ay4kxa1n+dLmttUlnP3V9oNE4hmsM=",
    version = "v0.6.6",
)

go_repository(
    name = "com_github_hashicorp_go_rootcerts",
    importpath = "github.com/hashicorp/go-rootcerts",
    sum = "h1:DMo4fmknnz0E0evoNYnV48RjWndOsmd6OW+09R3cEP8=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_hashicorp_go_sockaddr",
    importpath = "github.com/hashicorp/go-sockaddr",
    sum = "h1:ztczhD1jLxIRjVejw8gFomI1BQZOe2WoVOu0SyteCQc=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_hashicorp_go_syslog",
    importpath = "github.com/hashicorp/go-syslog",
    sum = "h1:KaodqZuhUoZereWVIYmpUgZysurB1kBLX2j0MwMrUAE=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_go_uuid",
    importpath = "github.com/hashicorp/go-uuid",
    sum = "h1:fv1ep09latC32wFoVwnqcnKJGnMSdBanPczbHAYm1BE=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_hashicorp_go_version",
    importpath = "github.com/hashicorp/go-version",
    sum = "h1:3vNe/fWF5CBgRIguda1meWhsZHy3m8gCJ5wx+dIzX/E=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    importpath = "github.com/hashicorp/golang-lru",
    sum = "h1:YDjusn29QI/Das2iO9M0BHnIbxPeyuCHsjMW+lJfyTc=",
    version = "v0.5.4",
)

go_repository(
    name = "com_github_hashicorp_hcl",
    importpath = "github.com/hashicorp/hcl",
    sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_logutils",
    importpath = "github.com/hashicorp/logutils",
    sum = "h1:dLEQVugN8vlakKOUE3ihGLTZJRB4j+M2cdTm/ORI65Y=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_mdns",
    importpath = "github.com/hashicorp/mdns",
    sum = "h1:WhIgCr5a7AaVH6jPUwjtRuuE7/RDufnUvzIr48smyxs=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_hashicorp_memberlist",
    importpath = "github.com/hashicorp/memberlist",
    sum = "h1:EmmoJme1matNzb+hMpDuR/0sbJSUisxyqBGG676r31M=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_hashicorp_serf",
    importpath = "github.com/hashicorp/serf",
    sum = "h1:YZ7UKsJv+hKjqGVUUbtE3HNj79Eln2oQ75tniF6iPt0=",
    version = "v0.8.2",
)

go_repository(
    name = "com_github_hashicorp_vault_api",
    importpath = "github.com/hashicorp/vault/api",
    sum = "h1:j08Or/wryXT4AcHj1oCbMd7IijXcKzYUGw59LGu9onU=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_hashicorp_vault_sdk",
    importpath = "github.com/hashicorp/vault/sdk",
    sum = "h1:mOEPeOhT7jl0J4AMl1E705+BcmeRs1VmKNb9F0sMLy8=",
    version = "v0.1.13",
)

go_repository(
    name = "com_github_hashicorp_yamux",
    importpath = "github.com/hashicorp/yamux",
    sum = "h1:kJCB4vdITiW1eC1vq2e6IsrXKrZit1bv/TDYFGMp4BQ=",
    version = "v0.0.0-20181012175058-2f1d1f20f75d",
)

go_repository(
    name = "com_github_howeyc_gopass",
    importpath = "github.com/howeyc/gopass",
    sum = "h1:kQWxfPIHVLbgLzphqk3QUflDy9QdksZR4ygR807bpy0=",
    version = "v0.0.0-20170109162249-bf9dde6d0d2c",
)

go_repository(
    name = "com_github_hpcloud_tail",
    importpath = "github.com/hpcloud/tail",
    sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_huandu_xstrings",
    importpath = "github.com/huandu/xstrings",
    sum = "h1:yPeWdRnmynF7p+lLYz0H2tthW9lqhMJrQV/U7yy4wX0=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_ianlancetaylor_demangle",
    importpath = "github.com/ianlancetaylor/demangle",
    sum = "h1:UDMh68UUwekSh5iP2OMhRRZJiiBccgV7axzUG8vi56c=",
    version = "v0.0.0-20181102032728-5e5cf60278f6",
)

go_repository(
    name = "com_github_imdario_mergo",
    importpath = "github.com/imdario/mergo",
    sum = "h1:b6R2BslTbIEToALKP7LxUvijTsNI9TAe80pLWN2g/HU=",
    version = "v0.3.12",
)

go_repository(
    name = "com_github_inconshreveable_mousetrap",
    importpath = "github.com/inconshreveable/mousetrap",
    sum = "h1:Z8tu5sraLXCXIcARxBp/8cbvlwVa7Z1NHg9XEKhtSvM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_influxdata_influxdb",
    importpath = "github.com/influxdata/influxdb",
    sum = "h1:AciJ2ei/llFRundm7CtqwF6B2aOds1A7QG3sMW8QiaQ=",
    version = "v0.0.0-20161215172503-049f9b42e9a5",
)

go_repository(
    name = "com_github_influxdata_tdigest",
    importpath = "github.com/influxdata/tdigest",
    sum = "h1:MHTrDWmQpHq/hkq+7cw9oYAt2PqUw52TZazRA0N7PGE=",
    version = "v0.0.0-20181121200506-bf2b5ad3c0a9",
)

go_repository(
    name = "com_github_jarcoal_httpmock",
    importpath = "github.com/jarcoal/httpmock",
    sum = "h1:cHtVEcTxRSX4J0je7mWPfc9BpDpqzXSJ5HbymZmyHck=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_jbenet_go_context",
    importpath = "github.com/jbenet/go-context",
    sum = "h1:BQSFePA1RWJOlocH6Fxy8MmwDt+yVQYULKfN0RoTN8A=",
    version = "v0.0.0-20150711004518-d14ea06fba99",
)

go_repository(
    name = "com_github_jcmturner_gofork",
    importpath = "github.com/jcmturner/gofork",
    sum = "h1:J7uCkflzTEhUZ64xqKnkDxq3kzc96ajM1Gli5ktUem8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_jenkins_x_go_scm",
    importpath = "github.com/jenkins-x/go-scm",
    sum = "h1:D7d1sDWUU+xocCNLQVoYKpMjVKnQvsPva+hPzruchbM=",
    version = "v1.5.117",
)

go_repository(
    name = "com_github_jessevdk_go_flags",
    importpath = "github.com/jessevdk/go-flags",
    sum = "h1:1jKYvbxEjfUl0fmqTCOfonvskHHXMjBySTLW4y9LFvc=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_jetstack_cert_manager",
    importpath = "github.com/jetstack/cert-manager",
    sum = "h1:gEhBV9I83m+kpQShDhNO4+J8O2qfNDjvAEL27pThGmg=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_jingyugao_rowserrcheck",
    importpath = "github.com/jingyugao/rowserrcheck",
    sum = "h1:GmsqmapfzSJkm28dhRoHz2tLRbJmqhU86IPgBtN3mmk=",
    version = "v0.0.0-20191204022205-72ab7603b68a",
)

go_repository(
    name = "com_github_jinzhu_gorm",
    importpath = "github.com/jinzhu/gorm",
    sum = "h1:Drgk1clyWT9t9ERbzHza6Mj/8FY/CqMyVzOiHviMo6Q=",
    version = "v1.9.12",
)

go_repository(
    name = "com_github_jinzhu_inflection",
    importpath = "github.com/jinzhu/inflection",
    sum = "h1:K317FqzuhWc8YvSVlFMCCUb36O/S9MCKRDI7QkRKD/E=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_jinzhu_now",
    importpath = "github.com/jinzhu/now",
    sum = "h1:g39TucaRWyV3dwDO++eEc6qf8TVIQ/Da48WmqjZ3i7E=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_jirfag_go_printf_func_name",
    importpath = "github.com/jirfag/go-printf-func-name",
    sum = "h1:KA9BjwUk7KlCh6S9EAGWBt1oExIUv9WyNCiRz5amv48=",
    version = "v0.0.0-20200119135958-7558a9eaa5af",
)

go_repository(
    name = "com_github_jmespath_go_jmespath",
    importpath = "github.com/jmespath/go-jmespath",
    sum = "h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_jmespath_go_jmespath_internal_testify",
    importpath = "github.com/jmespath/go-jmespath/internal/testify",
    sum = "h1:shLQSRRSCCPj3f2gpwzGwWFoC7ycTf1rcQZHOlsJ6N8=",
    version = "v1.5.1",
)

go_repository(
    name = "com_github_jmoiron_sqlx",
    importpath = "github.com/jmoiron/sqlx",
    sum = "h1:lrdPtrORjGv1HbbEvKWDUAy97mPpFm4B8hp77tcCUJY=",
    version = "v1.2.1-0.20190826204134-d7d95172beb5",
)

go_repository(
    name = "com_github_joefitzgerald_rainbow_reporter",
    importpath = "github.com/joefitzgerald/rainbow-reporter",
    sum = "h1:AuMG652zjdzI0YCCnXAqATtRBpGXMcAnrajcaTrSeuo=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_joho_godotenv",
    importpath = "github.com/joho/godotenv",
    sum = "h1:Zjp+RcGpHhGlrMbJzXTrZZPrWj+1vfm90La1wgB6Bhc=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_jonboulle_clockwork",
    importpath = "github.com/jonboulle/clockwork",
    sum = "h1:VKV+ZcuP6l3yW9doeqz6ziZGgcynBVQO+obU0+0hcPo=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_joshdk_go_junit",
    importpath = "github.com/joshdk/go-junit",
    sum = "h1:Bp5LAZasx/ev9wUmIIC74+MsXgwD99VjV1JmDVbpJm8=",
    version = "v0.0.0-20190428045703-ad7e11aa49ff",
)

go_repository(
    name = "com_github_jpillora_backoff",
    importpath = "github.com/jpillora/backoff",
    sum = "h1:K//n/AqR5HjG3qxbrBCL4vJPW0MVFSs9CPK1OOJdRME=",
    version = "v0.0.0-20180909062703-3050d21c67d7",
)

go_repository(
    name = "com_github_json_iterator_go",
    importpath = "github.com/json-iterator/go",
    sum = "h1:Kz6Cvnvv2wGdaG/V8yMvfkmNiXq9Ya2KUv4rouJJr68=",
    version = "v1.1.10",
)

go_repository(
    name = "com_github_jstemmer_go_junit_report",
    importpath = "github.com/jstemmer/go-junit-report",
    sum = "h1:6QPYqodiu3GuPL+7mfx+NwDdp2eTkp9IfEUpgAwUN0o=",
    version = "v0.9.1",
)

go_repository(
    name = "com_github_jtolds_gls",
    importpath = "github.com/jtolds/gls",
    sum = "h1:xdiiI2gbIgH/gLH7ADydsJ1uDOEzR8yvV7C0MuV77Wo=",
    version = "v4.20.0+incompatible",
)

go_repository(
    name = "com_github_julienschmidt_httprouter",
    importpath = "github.com/julienschmidt/httprouter",
    sum = "h1:TDTW5Yz1mjftljbcKqRcrYhd4XeOoI98t+9HbQbYf7g=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_kballard_go_shellquote",
    importpath = "github.com/kballard/go-shellquote",
    sum = "h1:Z9n2FFNUXsshfwJMBgNA0RU6/i7WVaAegv3PtuIHPMs=",
    version = "v0.0.0-20180428030007-95032a82bc51",
)

go_repository(
    name = "com_github_kelseyhightower_envconfig",
    importpath = "github.com/kelseyhightower/envconfig",
    sum = "h1:Im6hONhd3pLkfDFsbRgu68RDNkGF1r3dvMUtDTo2cv8=",
    version = "v1.4.0",
)

go_repository(
    name = "com_github_kevinburke_ssh_config",
    importpath = "github.com/kevinburke/ssh_config",
    sum = "h1:DowS9hvgyYSX4TO5NpyC606/Z4SxnNYbT+WX27or6Ck=",
    version = "v0.0.0-20201106050909-4977a11b4351",
)

go_repository(
    name = "com_github_kisielk_errcheck",
    importpath = "github.com/kisielk/errcheck",
    sum = "h1:reN85Pxc5larApoH1keMBiu2GWtPqXQ1nc9gx+jOU+E=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_kisielk_gotool",
    importpath = "github.com/kisielk/gotool",
    sum = "h1:AV2c/EiW3KqPNT9ZKl07ehoAGi4C5/01Cfbblndcapg=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_klauspost_compress",
    importpath = "github.com/klauspost/compress",
    sum = "h1:Znfn6hXZAHaLPNnlqUYRrBSReFHYybslgv4PTiyz6P0=",
    version = "v1.10.2",
)

go_repository(
    name = "com_github_klauspost_cpuid",
    importpath = "github.com/klauspost/cpuid",
    sum = "h1:1xAgYebNnsb9LKCdLOvFWtAxGU/33mjJtyOVbmUa0Us=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_klauspost_pgzip",
    importpath = "github.com/klauspost/pgzip",
    sum = "h1:oIPZROsWuPHpOdMVWLuJZXwgjhrW8r1yEX8UqMyeNHM=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_knative_build",
    importpath = "github.com/knative/build",
    sum = "h1:o/VYWA3HKyZlNqdU2hDE5LHpanBe8gazgPKL97XJ6bo=",
    version = "v0.1.2",
)

go_repository(
    name = "com_github_konsorten_go_windows_terminal_sequences",
    importpath = "github.com/konsorten/go-windows-terminal-sequences",
    sum = "h1:CE8S1cTafDpPvMhIxNJKvHsGVBgn1xWYf1NbHQhywc8=",
    version = "v1.0.3",
)

go_repository(
    name = "com_github_kr_logfmt",
    importpath = "github.com/kr/logfmt",
    sum = "h1:T+h1c/A9Gawja4Y9mFVWj2vyii2bbUNDw3kt9VxK2EY=",
    version = "v0.0.0-20140226030751-b84e30acd515",
)

go_repository(
    name = "com_github_kr_pretty",
    importpath = "github.com/kr/pretty",
    sum = "h1:Fmg33tUaq4/8ym9TJN1x7sLJnHVwhP33CNkpYV/7rwI=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_kr_pty",
    importpath = "github.com/kr/pty",
    sum = "h1:AkaSdXYQOWeaO3neb8EM634ahkXXe3jYbVh/F9lq+GI=",
    version = "v1.1.8",
)

go_repository(
    name = "com_github_kr_text",
    importpath = "github.com/kr/text",
    sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_lib_pq",
    importpath = "github.com/lib/pq",
    sum = "h1:LXpIM/LZ5xGFhOpXAQUIMM1HdyqzVYM13zNdjCEEcA0=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_liggitt_tabwriter",
    importpath = "github.com/liggitt/tabwriter",
    sum = "h1:9TO3cAIGXtEhnIaL+V+BEER86oLrvS+kWobKpbJuye0=",
    version = "v0.0.0-20181228230101-89fcab3d43de",
)

go_repository(
    name = "com_github_lightstep_tracecontext_go",
    importpath = "github.com/lightstep/tracecontext.go",
    sum = "h1:+2b6iGRJe3hvV/yVXrd41yVEjxuFHxasJqDhkIjS4gk=",
    version = "v0.0.0-20181129014701-1757c391b1ac",
)

go_repository(
    name = "com_github_lithammer_dedent",
    importpath = "github.com/lithammer/dedent",
    sum = "h1:VNzHMVCBNG1j0fh3OrsFRkVUwStdDArbgBWoPAffktY=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_logrusorgru_aurora",
    importpath = "github.com/logrusorgru/aurora",
    sum = "h1:9MlwzLdW7QSDrhDjFlsEYmxpFyIoXmYRon3dt0io31k=",
    version = "v0.0.0-20181002194514-a7b3b318ed4e",
)

go_repository(
    name = "com_github_lyft_protoc_gen_validate",
    importpath = "github.com/lyft/protoc-gen-validate",
    sum = "h1:KNt/RhmQTOLr7Aj8PsJ7mTronaFyx80mRTT9qF261dA=",
    version = "v0.0.13",
)

go_repository(
    name = "com_github_magiconair_properties",
    importpath = "github.com/magiconair/properties",
    sum = "h1:ZC2Vc7/ZFkGmsVC9KvOjumD+G5lXy2RtTKyzRKO2BQ4=",
    version = "v1.8.1",
)

go_repository(
    name = "com_github_mailru_easyjson",
    importpath = "github.com/mailru/easyjson",
    sum = "h1:jcoUdG1TzY/M/eM5BLFLP8DJeMximx5NQYSlLL9YeWc=",
    version = "v0.7.1-0.20191009090205-6c0755d89d1e",
)

go_repository(
    name = "com_github_makenowjust_heredoc",
    importpath = "github.com/MakeNowJust/heredoc",
    sum = "h1:sjQovDkwrZp8u+gxLtPgKGjk5hCxuy2hrRejBTA9xFU=",
    version = "v0.0.0-20170808103936-bb23615498cd",
)

go_repository(
    name = "com_github_maratori_testpackage",
    importpath = "github.com/maratori/testpackage",
    sum = "h1:QtJ5ZjqapShm0w5DosRjg0PRlSdAdlx+W6cCKoALdbQ=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_markbates_inflect",
    importpath = "github.com/markbates/inflect",
    sum = "h1:5fh1gzTFhfae06u3hzHYO9xe3l3v3nW5Pwt3naLTP5g=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_marstr_guid",
    importpath = "github.com/marstr/guid",
    sum = "h1:/M4H/1G4avsieL6BbUwCOBzulmoeKVP5ux/3mQNnbyI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_masterminds_goutils",
    importpath = "github.com/Masterminds/goutils",
    sum = "h1:zukEsf/1JZwCMgHiK3GZftabmxiCw4apj3a28RPBiVg=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_masterminds_semver",
    importpath = "github.com/Masterminds/semver",
    sum = "h1:H65muMkzWKEuNDnfl9d70GUjFniHKHRbFPGBuZ3QEww=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_masterminds_semver_v3",
    importpath = "github.com/Masterminds/semver/v3",
    sum = "h1:Y2lUDsFKVRSYGojLJ1yLxSXdMmMYTYls0rCvoqmMUQk=",
    version = "v3.1.0",
)

go_repository(
    name = "com_github_masterminds_sprig_v3",
    importpath = "github.com/Masterminds/sprig/v3",
    sum = "h1:wz22D0CiSctrliXiI9ZO3HoNApweeRGftyDN+BQa3B8=",
    version = "v3.0.2",
)

go_repository(
    name = "com_github_masterminds_vcs",
    importpath = "github.com/Masterminds/vcs",
    sum = "h1:NL3G1X7/7xduQtA2sJLpVpfHTNBALVNSjob6KEjPXNQ=",
    version = "v1.13.1",
)

go_repository(
    name = "com_github_matoous_godox",
    importpath = "github.com/matoous/godox",
    sum = "h1:RHba4YImhrUVQDHUCe2BNSOz4tVy2yGyXhvYDvxGgeE=",
    version = "v0.0.0-20190911065817-5d6d842e92eb",
)

go_repository(
    name = "com_github_mattbaird_jsonpatch",
    importpath = "github.com/mattbaird/jsonpatch",
    sum = "h1:+J2gw7Bw77w/fbK7wnNJJDKmw1IbWft2Ul5BzrG1Qm8=",
    version = "v0.0.0-20171005235357-81af80346b1a",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    importpath = "github.com/mattn/go-colorable",
    sum = "h1:6Su7aK7lXmJ/U79bYtBjLNaha4Fs1Rg9plHpcH+vvnE=",
    version = "v0.1.6",
)

go_repository(
    name = "com_github_mattn_go_ieproxy",
    importpath = "github.com/mattn/go-ieproxy",
    sum = "h1:qiyop7gCflfhwCzGyeT0gro3sF9AIg9HU98JORTkqfI=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    sum = "h1:wuysRhFDzyxgEmMf5xjvJ2M9dZoWAXNNr5LSBS7uHXY=",
    version = "v0.0.12",
)

go_repository(
    name = "com_github_mattn_go_runewidth",
    importpath = "github.com/mattn/go-runewidth",
    sum = "h1:3tS41NlGYSmhhe/8fhGRzc+z3AYCw1Fe1WAyLuujKs0=",
    version = "v0.0.8",
)

go_repository(
    name = "com_github_mattn_go_shellwords",
    importpath = "github.com/mattn/go-shellwords",
    sum = "h1:Y7Xqm8piKOO3v10Thp7Z36h4FYFjt5xB//6XvOrs2Gw=",
    version = "v1.0.10",
)

go_repository(
    name = "com_github_mattn_go_sqlite3",
    importpath = "github.com/mattn/go-sqlite3",
    sum = "h1:xQ15muvnzGBHpIpdrNi1DA5x0+TcBZzsIDwmw9uTHzw=",
    version = "v2.0.1+incompatible",
)

go_repository(
    name = "com_github_mattn_go_zglob",
    importpath = "github.com/mattn/go-zglob",
    sum = "h1:0qT24o2wsZ8cOXQAERwBX6s+rPMs/bJTKxLVVtgfDXc=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_mattn_goveralls",
    importpath = "github.com/mattn/goveralls",
    sum = "h1:7eJB6EqsPhRVxvwEXGnqdO2sJI0PTsrWoTMXEk9/OQc=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
    sum = "h1:I0XW9+e1XWDxdcEniV4rQAIOPUGDq67JSCiRCgGCZLI=",
    version = "v1.0.2-0.20181231171920-c182affec369",
)

go_repository(
    name = "com_github_maxbrunsfeld_counterfeiter_v6",
    importpath = "github.com/maxbrunsfeld/counterfeiter/v6",
    sum = "h1:g+4J5sZg6osfvEfkRZxJ1em0VT95/UOZgi/l7zi1/oE=",
    version = "v6.2.2",
)

go_repository(
    name = "com_github_mgutz_ansi",
    importpath = "github.com/mgutz/ansi",
    sum = "h1:j7+1HpAFS1zy5+Q4qx1fWh90gTKwiN4QCGoY9TWyyO4=",
    version = "v0.0.0-20170206155736-9520e82c474b",
)

go_repository(
    name = "com_github_mholt_archiver_v3",
    importpath = "github.com/mholt/archiver/v3",
    sum = "h1:vWjhY8SQp5yzM9P6OJ/eZEkmi3UAbRrxCq48MxjAzig=",
    version = "v3.3.0",
)

go_repository(
    name = "com_github_microsoft_go_winio",
    importpath = "github.com/Microsoft/go-winio",
    sum = "h1:FtSW/jqD+l4ba5iPBj9CODVtgfYAD8w2wS923g/cFDk=",
    version = "v0.4.16",
)

go_repository(
    name = "com_github_microsoft_hcsshim",
    importpath = "github.com/Microsoft/hcsshim",
    sum = "h1:ptnOoufxGSzauVTsdE+wMYnCWA301PdoN4xg5oRdZpg=",
    version = "v0.8.7",
)

go_repository(
    name = "com_github_miekg_dns",
    importpath = "github.com/miekg/dns",
    sum = "h1:sJFOl9BgwbYAWOGEwr61FU28pqsBNdpRBnhGXtO06Oo=",
    version = "v1.1.31",
)

go_repository(
    name = "com_github_mitchellh_cli",
    importpath = "github.com/mitchellh/cli",
    sum = "h1:iGBIsUe3+HZ/AD/Vd7DErOt5sU9fa8Uj7A2s1aggv1Y=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_copystructure",
    importpath = "github.com/mitchellh/copystructure",
    sum = "h1:Laisrj+bAB6b/yJwB5Bt3ITZhGJdqmxquMKeZ+mmkFQ=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_go_homedir",
    importpath = "github.com/mitchellh/go-homedir",
    sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_mitchellh_go_ps",
    importpath = "github.com/mitchellh/go-ps",
    sum = "h1:9+ke9YJ9KGWw5ANXK6ozjoK47uI3uNbXv4YVINBnGm8=",
    version = "v0.0.0-20190716172923-621e5597135b",
)

go_repository(
    name = "com_github_mitchellh_go_testing_interface",
    importpath = "github.com/mitchellh/go-testing-interface",
    sum = "h1:fzU/JVNcaqHQEcVFAKeR41fkiLdIPrefOvVG1VZ96U0=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_go_wordwrap",
    importpath = "github.com/mitchellh/go-wordwrap",
    sum = "h1:6GlHJ/LTGMrIJbwgdqdl2eEH8o+Exx/0m8ir9Gns0u4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_gox",
    importpath = "github.com/mitchellh/gox",
    sum = "h1:lfGJxY7ToLJQjHHwi0EX6uYBdK78egf954SQl13PQJc=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_mitchellh_iochan",
    importpath = "github.com/mitchellh/iochan",
    sum = "h1:C+X3KsSTLFVBr/tK1eYN/vs4rJcvsiLU338UhYPJWeY=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_ioprogress",
    importpath = "github.com/mitchellh/ioprogress",
    sum = "h1:Qa6dnn8DlasdXRnacluu8HzPts0S1I9zvvUPDbBnXFI=",
    version = "v0.0.0-20180201004757-6a23b12fa88e",
)

go_repository(
    name = "com_github_mitchellh_mapstructure",
    importpath = "github.com/mitchellh/mapstructure",
    sum = "h1:cCBH2gTD2K0OtLlv/Y5H01VQCqmlDxz30kS5Y5bqfLA=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_mitchellh_osext",
    importpath = "github.com/mitchellh/osext",
    sum = "h1:2+myh5ml7lgEU/51gbeLHfKGNfgEQQIWrlbdaOsidbQ=",
    version = "v0.0.0-20151018003038-5e2d6d41470f",
)

go_repository(
    name = "com_github_mitchellh_reflectwalk",
    importpath = "github.com/mitchellh/reflectwalk",
    sum = "h1:9D+8oIskB4VJBN5SFlmc27fSlIBZaov1Wpk/IfikLNY=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_moby_term",
    importpath = "github.com/moby/term",
    sum = "h1:aY7OQNf2XqY/JQ6qREWamhI/81os/agb2BAGpcx5yWI=",
    version = "v0.0.0-20200312100748-672ec06f55cd",
)

go_repository(
    name = "com_github_modern_go_concurrent",
    importpath = "github.com/modern-go/concurrent",
    sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
    version = "v0.0.0-20180306012644-bacd9c7ef1dd",
)

go_repository(
    name = "com_github_modern_go_reflect2",
    importpath = "github.com/modern-go/reflect2",
    sum = "h1:9f412s+6RmYXLWZSEzVVgPGK7C2PphHj5RJrvfx9AWI=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_mohae_deepcopy",
    importpath = "github.com/mohae/deepcopy",
    sum = "h1:RWengNIwukTxcDr9M+97sNutRR1RKhG96O6jWumTTnw=",
    version = "v0.0.0-20170929034955-c48cc78d4826",
)

go_repository(
    name = "com_github_morikuni_aec",
    importpath = "github.com/morikuni/aec",
    sum = "h1:nP9CBfwrvYnBRgY6qfDQkygYDmYwOilePFkwzv4dU8A=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mozilla_tls_observatory",
    importpath = "github.com/mozilla/tls-observatory",
    sum = "h1:1xJ+Xi9lYWLaaP4yB67ah0+548CD3110mCPWhVVjFkI=",
    version = "v0.0.0-20200317151703-4fa42e1c2dee",
)

go_repository(
    name = "com_github_munnerz_crd_schema_fuzz",
    importpath = "github.com/munnerz/crd-schema-fuzz",
    sum = "h1:8erI9yzEnOGw9K5O+a8zZdoo8N/OwrFi7c7SjBtkHAs=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_munnerz_goautoneg",
    importpath = "github.com/munnerz/goautoneg",
    sum = "h1:C3w9PqII01/Oq1c1nUAm88MOHcQC9l5mIlSMApZMrHA=",
    version = "v0.0.0-20191010083416-a7dc8b61c822",
)

go_repository(
    name = "com_github_mwitkow_go_conntrack",
    importpath = "github.com/mwitkow/go-conntrack",
    sum = "h1:KUppIJq7/+SVif2QVs3tOP0zanoHgBEVAwHxUSIzRqU=",
    version = "v0.0.0-20190716064945-2f068394615f",
)

go_repository(
    name = "com_github_mxk_go_flowrate",
    importpath = "github.com/mxk/go-flowrate",
    sum = "h1:y5//uYreIhSUg3J1GEMiLbxo1LJaP8RfCpH6pymGZus=",
    version = "v0.0.0-20140419014527-cca7078d478f",
)

go_repository(
    name = "com_github_nakabonne_nestif",
    importpath = "github.com/nakabonne/nestif",
    sum = "h1:+yOViDGhg8ygGrmII72nV9B/zGxY188TYpfolntsaPw=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_natefinch_lumberjack",
    importpath = "github.com/natefinch/lumberjack",
    sum = "h1:4QJd3OLAMgj7ph+yZTuX13Ld4UpgHp07nNdFX7mqFfM=",
    version = "v2.0.0+incompatible",
)

go_repository(
    name = "com_github_nats_io_gnatsd",
    importpath = "github.com/nats-io/gnatsd",
    sum = "h1:RconcfDeWpKCD6QIIwiVFcvForlXpWeJP7i5/lDLy44=",
    version = "v1.4.1",
)

go_repository(
    name = "com_github_nats_io_go_nats",
    importpath = "github.com/nats-io/go-nats",
    sum = "h1:oQOfHcLr8hb43QG8yeVyY2jtarIaTjOv41CGdF3tTvQ=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_nats_io_jwt",
    importpath = "github.com/nats-io/jwt",
    sum = "h1:+RB5hMpXUUA2dfxuhBTEkMOrYmM+gKIZYS1KjSostMI=",
    version = "v0.3.2",
)

go_repository(
    name = "com_github_nats_io_nats_go",
    importpath = "github.com/nats-io/nats.go",
    sum = "h1:ik3HbLhZ0YABLto7iX80pZLPw/6dx3T+++MZJwLnMrQ=",
    version = "v1.9.1",
)

go_repository(
    name = "com_github_nats_io_nats_server_v2",
    importpath = "github.com/nats-io/nats-server/v2",
    sum = "h1:i2Ly0B+1+rzNZHHWtD4ZwKi+OU5l+uQo1iDHZ2PmiIc=",
    version = "v2.1.2",
)

go_repository(
    name = "com_github_nats_io_nkeys",
    importpath = "github.com/nats-io/nkeys",
    sum = "h1:6JrEfig+HzTH85yxzhSVbjHRJv9cn0p6n3IngIcM5/k=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_nats_io_nuid",
    importpath = "github.com/nats-io/nuid",
    sum = "h1:5iA8DT8V7q8WK2EScv2padNa/rTESc1KdnPw4TC2paw=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_nbio_st",
    importpath = "github.com/nbio/st",
    sum = "h1:W6apQkHrMkS0Muv8G/TipAy/FJl/rCYT0+EuS8+Z0z4=",
    version = "v0.0.0-20140626010706-e9e8d9816f32",
)

go_repository(
    name = "com_github_nbutton23_zxcvbn_go",
    importpath = "github.com/nbutton23/zxcvbn-go",
    sum = "h1:AREM5mwr4u1ORQBMvzfzBgpsctsbQikCVpvC+tX285E=",
    version = "v0.0.0-20180912185939-ae427f1e4c1d",
)

go_repository(
    name = "com_github_ncw_swift",
    importpath = "github.com/ncw/swift",
    sum = "h1:4DQRPj35Y41WogBxyhOXlrI37nzGlyEcsforeudyYPQ=",
    version = "v1.0.47",
)

go_repository(
    name = "com_github_niemeyer_pretty",
    importpath = "github.com/niemeyer/pretty",
    sum = "h1:fD57ERR4JtEqsWbfPhv4DMiApHyliiK5xCTNVSPiaAs=",
    version = "v0.0.0-20200227124842-a10e7caefd8e",
)

go_repository(
    name = "com_github_nwaples_rardecode",
    importpath = "github.com/nwaples/rardecode",
    sum = "h1:r7vGuS5akxOnR4JQSkko62RJ1ReCMXxQRPtxsiFMBOs=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_nxadm_tail",
    importpath = "github.com/nxadm/tail",
    sum = "h1:DQuhQpB1tVlglWS2hLQ5OV6B5r8aGxSrPc5Qo6uTN78=",
    version = "v1.4.4",
)

go_repository(
    name = "com_github_nytimes_gziphandler",
    importpath = "github.com/NYTimes/gziphandler",
    sum = "h1:lsxEuwrXEAokXB9qhlbKWPpo3KMLZQ5WB5WLQRW1uq0=",
    version = "v0.0.0-20170623195520-56545f4a5d46",
)

go_repository(
    name = "com_github_octago_sflags",
    importpath = "github.com/octago/sflags",
    sum = "h1:XceYzkRXGAHa/lSFmKLcaxSrsh4MTuOMQdIGsUD0wlk=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_oklog_run",
    importpath = "github.com/oklog/run",
    sum = "h1:Ru7dDtJNOyC66gQ5dQmaCa0qIsAUFY3sFpK1Xk8igrw=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_oklog_ulid",
    importpath = "github.com/oklog/ulid",
    sum = "h1:EGfNDEx6MqHz8B3uNV6QAib1UR2Lm97sHi3ocA6ESJ4=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_olekukonko_tablewriter",
    importpath = "github.com/olekukonko/tablewriter",
    sum = "h1:vHD/YYe1Wolo78koG299f7V/VAS08c6IpCLn+Ejf/w8=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_oneofone_xxhash",
    importpath = "github.com/OneOfOne/xxhash",
    sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_onsi_ginkgo",
    importpath = "github.com/onsi/ginkgo",
    sum = "h1:jMU0WaQrP0a/YAEq8eJmJKjBoMs+pClEr1vDMlM/Do4=",
    version = "v1.14.1",
)

go_repository(
    name = "com_github_onsi_gomega",
    importpath = "github.com/onsi/gomega",
    sum = "h1:aY/nuoWlKJud2J6U0E3NWsjlg+0GtwXxgEqthRdzlcs=",
    version = "v1.10.2",
)

go_repository(
    name = "com_github_op_go_logging",
    importpath = "github.com/op/go-logging",
    sum = "h1:lDH9UUVJtmYCjyT0CI4q8xvlXPxeZ0gYCVvWbmPlp88=",
    version = "v0.0.0-20160315200505-970db520ece7",
)

go_repository(
    name = "com_github_opencontainers_go_digest",
    importpath = "github.com/opencontainers/go-digest",
    sum = "h1:WzifXhOVOEOuFYOJAW6aQqW0TooG2iki3E3Ii+WN7gQ=",
    version = "v1.0.0-rc1",
)

go_repository(
    name = "com_github_opencontainers_image_spec",
    importpath = "github.com/opencontainers/image-spec",
    sum = "h1:JMemWkRwHx4Zj+fVxWoMCFm/8sYGGrUVojFA6h/TRcI=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_opencontainers_runc",
    importpath = "github.com/opencontainers/runc",
    sum = "h1:GlxAyO6x8rfZYN9Tt0Kti5a/cP41iuiO2yYT0IJGY8Y=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_opencontainers_runtime_spec",
    importpath = "github.com/opencontainers/runtime-spec",
    sum = "h1:eNUVfm/RFLIi1G7flU5/ZRTHvd4kcVuzfRnL6OFlzCI=",
    version = "v0.1.2-0.20190507144316-5b71a03e2700",
)

go_repository(
    name = "com_github_opencontainers_runtime_tools",
    importpath = "github.com/opencontainers/runtime-tools",
    sum = "h1:H7DMc6FAjgwZZi8BRqjrAAHWoqEr5e5L6pS4V0ezet4=",
    version = "v0.0.0-20181011054405-1d69bd0f9c39",
)

go_repository(
    name = "com_github_openpeedeep_depguard",
    importpath = "github.com/OpenPeeDeeP/depguard",
    sum = "h1:VlW4R6jmBIv3/u1JNlawEvJMM4J+dPORPaZasQee8Us=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_openzipkin_zipkin_go",
    importpath = "github.com/openzipkin/zipkin-go",
    sum = "h1:nY8Hti+WKaP0cRsSeQ026wU03QsM762XBeCXBb9NAWI=",
    version = "v0.2.2",
)

go_repository(
    name = "com_github_otiai10_copy",
    importpath = "github.com/otiai10/copy",
    sum = "h1:DDNipYy6RkIkjMwy+AWzgKiNTyj2RUI9yEMeETEpVyc=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_otiai10_curr",
    importpath = "github.com/otiai10/curr",
    sum = "h1:+OLn68pqasWca0z5ryit9KGfp3sUsW4Lqg32iRMJyzs=",
    version = "v0.0.0-20150429015615-9b4961190c95",
)

go_repository(
    name = "com_github_otiai10_mint",
    importpath = "github.com/otiai10/mint",
    sum = "h1:Ady6MKVezQwHBkGzLFbrsywyp09Ah7rkmfjV3Bcr5uc=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_pascaldekloe_goe",
    importpath = "github.com/pascaldekloe/goe",
    sum = "h1:cBOtyMzM9HTpWjXfbbunk26uA6nG3a8n06Wieeh0MwY=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_pavel_v_chernykh_keystore_go",
    importpath = "github.com/pavel-v-chernykh/keystore-go",
    sum = "h1:Jd6xfriVlJ6hWPvYOE0Ni0QWcNTLRehfGPFxr3eSL80=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_pborman_uuid",
    importpath = "github.com/pborman/uuid",
    sum = "h1:J7Q5mO4ysT1dv8hyrUGHb9+ooztCXu1D8MY8DZYsu3g=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_pelletier_go_buffruneio",
    importpath = "github.com/pelletier/go-buffruneio",
    sum = "h1:U4t4R6YkofJ5xHm3dJzuRpPZ0mr5MMCoAWooScCR7aA=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    importpath = "github.com/pelletier/go-toml",
    sum = "h1:Keo9qb7iRJs2voHvunFtuuYFsbWeOBh8/P9v/kVMFtw=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_peterbourgon_diskv",
    importpath = "github.com/peterbourgon/diskv",
    sum = "h1:UBdAOUP5p4RWqPBg048CAvpKN+vxiaj6gdUUzhl4XmI=",
    version = "v2.0.1+incompatible",
)

go_repository(
    name = "com_github_phayes_checkstyle",
    importpath = "github.com/phayes/checkstyle",
    sum = "h1:CdDQnGF8Nq9ocOS/xlSptM1N3BbrA6/kmaep5ggwaIA=",
    version = "v0.0.0-20170904204023-bfd46e6a821d",
)

go_repository(
    name = "com_github_phayes_freeport",
    importpath = "github.com/phayes/freeport",
    sum = "h1:JhzVVoYvbOACxoUmOs6V/G4D5nPVUW73rKvXxP4XUJc=",
    version = "v0.0.0-20180830031419-95f893ade6f2",
)

go_repository(
    name = "com_github_pierrec_lz4",
    importpath = "github.com/pierrec/lz4",
    sum = "h1:6aCX4/YZ9v8q69hTyiR7dNLnTA3fgtKHVVW5BCd5Znw=",
    version = "v2.2.6+incompatible",
)

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
    version = "v0.9.1",
)

go_repository(
    name = "com_github_pkg_profile",
    importpath = "github.com/pkg/profile",
    sum = "h1:F++O52m40owAmADcojzM+9gyjmMOY/T4oYJkgFDH8RE=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_pmezard_go_difflib",
    importpath = "github.com/pmezard/go-difflib",
    sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_posener_complete",
    importpath = "github.com/posener/complete",
    sum = "h1:ccV59UEOTzVDnDUEFdT95ZzHVZ+5+158q8+SJb2QV5w=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_pquerna_cachecontrol",
    importpath = "github.com/pquerna/cachecontrol",
    sum = "h1:0XM1XL/OFFJjXsYXlG30spTkV/E9+gmd5GD1w2HE8xM=",
    version = "v0.0.0-20171018203845-0dec1b30a021",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    importpath = "github.com/prometheus/client_golang",
    sum = "h1:NTGy1Ja9pByO+xAeH/qiWnLrKtr3hJPNjaVUwnjpdpA=",
    version = "v1.7.1",
)

go_repository(
    name = "com_github_prometheus_client_model",
    importpath = "github.com/prometheus/client_model",
    sum = "h1:uq5h0d+GuxiXLJLNABMgp2qUWDPiLvgCzz2dUR+/W/M=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_prometheus_common",
    importpath = "github.com/prometheus/common",
    sum = "h1:RyRA7RzGXQZiW+tGMr7sxa85G1z0yOpM1qq5c8lNawc=",
    version = "v0.10.0",
)

go_repository(
    name = "com_github_prometheus_procfs",
    importpath = "github.com/prometheus/procfs",
    sum = "h1:F0+tqvhOksq22sc6iCHF5WGlWjdwj92p0udFh1VFBS8=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_prometheus_tsdb",
    importpath = "github.com/prometheus/tsdb",
    sum = "h1:YZcsG11NqnK4czYLrWd9mpEuAJIHVQLwdrleYfszMAA=",
    version = "v0.7.1",
)

go_repository(
    name = "com_github_puerkitobio_purell",
    importpath = "github.com/PuerkitoBio/purell",
    sum = "h1:WEQqlqaGbrPkxLJWfBwQmfEAE1Z7ONdDLqrN38tNFfI=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_puerkitobio_urlesc",
    importpath = "github.com/PuerkitoBio/urlesc",
    sum = "h1:d+Bc7a5rLufV/sSk/8dngufqelfh6jnri85riMAaF/M=",
    version = "v0.0.0-20170810143723-de5bf2ad4578",
)

go_repository(
    name = "com_github_quasilyte_go_consistent",
    importpath = "github.com/quasilyte/go-consistent",
    sum = "h1:JoUA0uz9U0FVFq5p4LjEq4C0VgQ0El320s3Ms0V4eww=",
    version = "v0.0.0-20190521200055-c6f3937de18c",
)

go_repository(
    name = "com_github_quasilyte_go_ruleguard",
    importpath = "github.com/quasilyte/go-ruleguard",
    sum = "h1:DvnesvLtRPQOvaUbfXfh0tpMHg29by0H7F2U+QIkSu8=",
    version = "v0.1.2-0.20200318202121-b00d7a75d3d8",
)

go_repository(
    name = "com_github_rcrowley_go_metrics",
    importpath = "github.com/rcrowley/go-metrics",
    sum = "h1:eUm8ma4+yPknhXtkYlWh3tMkE6gBjXZToDned9s2gbQ=",
    version = "v0.0.0-20190706150252-9beb055b7962",
)

go_repository(
    name = "com_github_rogpeppe_fastuuid",
    importpath = "github.com/rogpeppe/fastuuid",
    sum = "h1:Ppwyp6VYCF1nvBTXL3trRso7mXMlRrw9ooo375wvi2s=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_rogpeppe_go_internal",
    importpath = "github.com/rogpeppe/go-internal",
    sum = "h1:qLvObTrvO/XRCqmkKxUlOBc48bI3efyDuAZe25QiF0w=",
    version = "v1.5.2",
)

go_repository(
    name = "com_github_rubiojr_go_vhd",
    importpath = "github.com/rubiojr/go-vhd",
    sum = "h1:ht7N4d/B7Ezf58nvMNVF3OlvDlz9pp+WHVcRNS0nink=",
    version = "v0.0.0-20160810183302-0bfd3b39853c",
)

go_repository(
    name = "com_github_russross_blackfriday",
    importpath = "github.com/russross/blackfriday",
    sum = "h1:HyvC0ARfnZBqnXwABFeSZHpKvJHJJfPz81GNueLj0oo=",
    version = "v1.5.2",
)

go_repository(
    name = "com_github_russross_blackfriday_v2",
    importpath = "github.com/russross/blackfriday/v2",
    sum = "h1:lPqVAte+HuHNfhJ/0LC98ESWRz8afy9tM/0RK8m9o+Q=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_ryancurrah_gomodguard",
    importpath = "github.com/ryancurrah/gomodguard",
    sum = "h1:DWbye9KyMgytn8uYpuHkwf0RHqAYO6Ay/D0TbCpPtVU=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_ryanuber_columnize",
    importpath = "github.com/ryanuber/columnize",
    sum = "h1:j1Wcmh8OrK4Q7GXY+V7SVSY8nUWQxHW5TkBe7YUl+2s=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_ryanuber_go_glob",
    importpath = "github.com/ryanuber/go-glob",
    sum = "h1:iQh3xXAumdQ+4Ufa5b25cRpC5TYKlno6hsv6Cb3pkBk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_sassoftware_go_rpmutils",
    importpath = "github.com/sassoftware/go-rpmutils",
    sum = "h1:+gCnWOZV8Z/8jehJ2CdqB47Z3S+SREmQcuXkRFLNsiI=",
    version = "v0.0.0-20190420191620-a8f1baeba37b",
)

go_repository(
    name = "com_github_satori_go_uuid",
    importpath = "github.com/satori/go.uuid",
    sum = "h1:0uYX9dsZ2yD7q2RtLRtPSdGDWzjeM3TbMJP9utgA0ww=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_sclevine_spec",
    importpath = "github.com/sclevine/spec",
    sum = "h1:1Jwdf9jSfDl9NVmt8ndHqbTZ7XCCPbh1jI3hkDBHVYA=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_sean_seed",
    importpath = "github.com/sean-/seed",
    sum = "h1:nn5Wsu0esKSJiIVhscUtVbo7ada43DJhG55ua/hjS5I=",
    version = "v0.0.0-20170313163322-e2103e2c3529",
)

go_repository(
    name = "com_github_securego_gosec",
    importpath = "github.com/securego/gosec",
    sum = "h1:rq2/kILQnPtq5oL4+IAjgVOjh5e2yj2aaCYi7squEvI=",
    version = "v0.0.0-20200401082031-e946c8c39989",
)

go_repository(
    name = "com_github_securego_gosec_v2",
    importpath = "github.com/securego/gosec/v2",
    sum = "h1:y/9mCF2WPDbSDpL3QDWZD3HHGrSYw0QSHnCqTfs4JPE=",
    version = "v2.3.0",
)

go_repository(
    name = "com_github_sergi_go_diff",
    importpath = "github.com/sergi/go-diff",
    sum = "h1:we8PVUC3FE2uYfodKH/nBHMSetSfHDR6scGdBi+erh0=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_shirou_gopsutil",
    importpath = "github.com/shirou/gopsutil",
    sum = "h1:WokF3GuxBeL+n4Lk4Fa8v9mbdjlrl7bHuneF4N1bk2I=",
    version = "v0.0.0-20190901111213-e4ec7b275ada",
)

go_repository(
    name = "com_github_shirou_w32",
    importpath = "github.com/shirou/w32",
    sum = "h1:udFKJ0aHUL60LboW/A+DfgoHVedieIzIXE8uylPue0U=",
    version = "v0.0.0-20160930032740-bb4de0191aa4",
)

go_repository(
    name = "com_github_shopify_logrus_bugsnag",
    importpath = "github.com/Shopify/logrus-bugsnag",
    sum = "h1:UrqY+r/OJnIp5u0s1SbQ8dVfLCZJsnvazdBP5hS4iRs=",
    version = "v0.0.0-20171204204709-577dee27f20d",
)

go_repository(
    name = "com_github_shopify_sarama",
    importpath = "github.com/Shopify/sarama",
    sum = "h1:XxJBCZEoWJtoWjf/xRbmGUpAmTZGnuuF0ON0EvxxBrs=",
    version = "v1.23.1",
)

go_repository(
    name = "com_github_shopify_toxiproxy",
    importpath = "github.com/Shopify/toxiproxy",
    sum = "h1:TKdv8HiTLgE5wdJuEML90aBgNWsokNbMijUGhmcoBJc=",
    version = "v2.1.4+incompatible",
)

go_repository(
    name = "com_github_shurcool_githubv4",
    importpath = "github.com/shurcooL/githubv4",
    sum = "h1:Cocq9/ZZxCoiybhygOR7hX4E3/PkV8eNbd1AEcUvaHM=",
    version = "v0.0.0-20191102174205-af46314aec7b",
)

go_repository(
    name = "com_github_shurcool_go",
    importpath = "github.com/shurcooL/go",
    sum = "h1:MZM7FHLqUHYI0Y/mQAt3d2aYa0SiNms/hFqC9qJYolM=",
    version = "v0.0.0-20180423040247-9e1955d9fb6e",
)

go_repository(
    name = "com_github_shurcool_go_goon",
    importpath = "github.com/shurcooL/go-goon",
    sum = "h1:llrF3Fs4018ePo4+G/HV/uQUqEI1HMDjCeOf2V6puPc=",
    version = "v0.0.0-20170922171312-37c2f522c041",
)

go_repository(
    name = "com_github_shurcool_graphql",
    importpath = "github.com/shurcooL/graphql",
    sum = "h1:tygelZueB1EtXkPI6mQ4o9DQ0+FKW41hTbunoXZCTqk=",
    version = "v0.0.0-20181231061246-d48a9a75455f",
)

go_repository(
    name = "com_github_shurcool_sanitized_anchor_name",
    importpath = "github.com/shurcooL/sanitized_anchor_name",
    sum = "h1:PdmoCO6wvbs+7yrJyMORt4/BmY5IYyJwS/kOiWx8mHo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    importpath = "github.com/sirupsen/logrus",
    sum = "h1:UBcNElsrwanuuMsnGSlYmtmgbb23qDR5dG+6X6Oo89I=",
    version = "v1.6.0",
)

go_repository(
    name = "com_github_smartystreets_assertions",
    importpath = "github.com/smartystreets/assertions",
    sum = "h1:42S6lae5dvLc7BrLu/0ugRtcFVjoJNMC/N3yZFZkDFs=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_smartystreets_go_aws_auth",
    importpath = "github.com/smartystreets/go-aws-auth",
    sum = "h1:hp2CYQUINdZMHdvTdXtPOY2ainKl4IoMcpAXEf2xj3Q=",
    version = "v0.0.0-20180515143844-0c1422d1fdb9",
)

go_repository(
    name = "com_github_smartystreets_goconvey",
    importpath = "github.com/smartystreets/goconvey",
    sum = "h1:fv0U8FUIMPNf1L9lnHLvLhgicrIVChEkdzIKYqbNC9s=",
    version = "v1.6.4",
)

go_repository(
    name = "com_github_smartystreets_gunit",
    importpath = "github.com/smartystreets/gunit",
    sum = "h1:RyPDUFcJbvtXlhJPk7v+wnxZRY2EUokhEYl2EJOPToI=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_soheilhy_cmux",
    importpath = "github.com/soheilhy/cmux",
    sum = "h1:0HKaf1o97UwFjHH9o5XsHUOF+tqmdA7KEzXLpiyaw0E=",
    version = "v0.1.4",
)

go_repository(
    name = "com_github_sourcegraph_go_diff",
    importpath = "github.com/sourcegraph/go-diff",
    sum = "h1:lhIKJ2nXLZZ+AfbHpYxTn0pXpNTTui0DX7DO3xeb1Zs=",
    version = "v0.5.3",
)

go_repository(
    name = "com_github_spaolacci_murmur3",
    importpath = "github.com/spaolacci/murmur3",
    sum = "h1:qLC7fQah7D6K1B0ujays3HV9gkFtllcxhzImRR7ArPQ=",
    version = "v0.0.0-20180118202830-f09979ecbc72",
)

go_repository(
    name = "com_github_spf13_afero",
    importpath = "github.com/spf13/afero",
    sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_spf13_cast",
    importpath = "github.com/spf13/cast",
    sum = "h1:nFm6S0SMdyzrzcmThSipiEubIDy8WEXKNZ0UOgiRpng=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_spf13_cobra",
    importpath = "github.com/spf13/cobra",
    sum = "h1:6m/oheQuQ13N9ks4hubMG6BnvwOeaJrqSPLahSnczz8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_spf13_jwalterweatherman",
    importpath = "github.com/spf13/jwalterweatherman",
    sum = "h1:ue6voC5bR5F8YxI5S67j9i582FU4Qvo2bmqnqMYADFk=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_spf13_pflag",
    importpath = "github.com/spf13/pflag",
    sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_spf13_viper",
    importpath = "github.com/spf13/viper",
    sum = "h1:xVKxvI7ouOI5I+U9s2eeiUfMaWBVoXA3AWskkrqK0VM=",
    version = "v1.7.0",
)

go_repository(
    name = "com_github_src_d_gcfg",
    importpath = "github.com/src-d/gcfg",
    sum = "h1:xXbNR5AlLSA315x2UO+fTSSAXCDf+Ar38/6oyGbDKQ4=",
    version = "v1.4.0",
)

go_repository(
    name = "com_github_stackexchange_wmi",
    importpath = "github.com/StackExchange/wmi",
    sum = "h1:fLjPD/aNc3UIOA6tDi6QXUemppXK3P9BI7mr2hd6gx8=",
    version = "v0.0.0-20180116203802-5d049714c4a6",
)

go_repository(
    name = "com_github_stoewer_go_strcase",
    importpath = "github.com/stoewer/go-strcase",
    sum = "h1:Z2iHWqGXH00XYgqDmNgQbIBxf3wrNq0F3feEy0ainaU=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_streadway_amqp",
    importpath = "github.com/streadway/amqp",
    sum = "h1:0ngsPmuP6XIjiFRNFYlvKwSr5zff2v+uPHaffZ6/M4k=",
    version = "v0.0.0-20190404075320-75d898a42a94",
)

go_repository(
    name = "com_github_streadway_quantile",
    importpath = "github.com/streadway/quantile",
    sum = "h1:7z3LSn867ex6VSaahyKadf4WtSsJIgne6A1WLOAGM8A=",
    version = "v0.0.0-20150917103942-b0c588724d25",
)

go_repository(
    name = "com_github_stretchr_objx",
    importpath = "github.com/stretchr/objx",
    sum = "h1:Hbg2NidpLE8veEBkEZTL3CvlkUIVzuU9jDplZO54c48=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_stretchr_testify",
    importpath = "github.com/stretchr/testify",
    sum = "h1:hDPOHmpOpP40lSULcqw7IrRb/u7w6RpDC9399XyoNd0=",
    version = "v1.6.1",
)

go_repository(
    name = "com_github_subosito_gotenv",
    importpath = "github.com/subosito/gotenv",
    sum = "h1:Slr1R9HxAlEKefgq5jn9U+DnETlIUa6HfgEzj0g5d7s=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_syndtr_gocapability",
    importpath = "github.com/syndtr/gocapability",
    sum = "h1:zLV6q4e8Jv9EHjNg/iHfzwDkCve6Ua5jCygptrtXHvI=",
    version = "v0.0.0-20170704070218-db04d3cc01c8",
)

go_repository(
    name = "com_github_tdakkota_asciicheck",
    importpath = "github.com/tdakkota/asciicheck",
    sum = "h1:HxLVTlqcHhFAz3nWUcuvpH7WuOMv8LQoCWmruLfFH2U=",
    version = "v0.0.0-20200416200610-e657995f937b",
)

go_repository(
    name = "com_github_tektoncd_pipeline",
    importpath = "github.com/tektoncd/pipeline",
    sum = "h1:hWdWj5bDjkSGYLlJS+u+Kh9ZktBJgs2JNUv/kP0LVOA=",
    version = "v0.13.1-0.20200625065359-44f22a067b75",
)

go_repository(
    name = "com_github_tektoncd_plumbing",
    importpath = "github.com/tektoncd/plumbing",
    sum = "h1:crv70CBAJ2gZFSbf13aRVwdbjR2GYwTms/ZEok/SnFM=",
    version = "v0.0.0-20200430135134-e53521e1d887",
)

go_repository(
    name = "com_github_tektoncd_plumbing_pipelinerun_logs",
    importpath = "github.com/tektoncd/plumbing/pipelinerun-logs",
    sum = "h1:9qeyrQsoPZbHOyOPt0OeB1TCYXfYb5swrxlFWzTIYYk=",
    version = "v0.0.0-20191206114338-712d544c2c21",
)

go_repository(
    name = "com_github_tetafro_godot",
    importpath = "github.com/tetafro/godot",
    sum = "h1:Dib7un+rYJFUi8vN0Bk6EHheKy6fv6ZzFURHw75g6m8=",
    version = "v0.4.2",
)

go_repository(
    name = "com_github_tidwall_pretty",
    importpath = "github.com/tidwall/pretty",
    sum = "h1:HsD+QiTn7sK6flMKIvNmpqz1qrpP3Ps6jOKIKMooyg4=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_timakin_bodyclose",
    importpath = "github.com/timakin/bodyclose",
    sum = "h1:ig99OeTyDwQWhPe2iw9lwfQVF1KB3Q4fpP3X7/2VBG8=",
    version = "v0.0.0-20200424151742-cb6215831a94",
)

go_repository(
    name = "com_github_tj_assert",
    importpath = "github.com/tj/assert",
    sum = "h1:Rw8kxzWo1mr6FSaYXjQELRe88y2KdfynXdnK72rdjtA=",
    version = "v0.0.0-20171129193455-018094318fb0",
)

go_repository(
    name = "com_github_tj_go_elastic",
    importpath = "github.com/tj/go-elastic",
    sum = "h1:eGaGNxrtoZf/mBURsnNQKDR7u50Klgcf2eFDQEnc8Bc=",
    version = "v0.0.0-20171221160941-36157cbbebc2",
)

go_repository(
    name = "com_github_tj_go_kinesis",
    importpath = "github.com/tj/go-kinesis",
    sum = "h1:m74UWYy+HBs+jMFR9mdZU6shPewugMyH5+GV6LNgW8w=",
    version = "v0.0.0-20171128231115-08b17f58cb1b",
)

go_repository(
    name = "com_github_tj_go_spin",
    importpath = "github.com/tj/go-spin",
    sum = "h1:lhdWZsvImxvZ3q1C5OIB7d72DuOwP4O2NdBg9PyzNds=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_tmc_grpc_websocket_proxy",
    importpath = "github.com/tmc/grpc-websocket-proxy",
    sum = "h1:LnC5Kc/wtumK+WB441p7ynQJzVuNRJiqddSIE3IlSEQ=",
    version = "v0.0.0-20190109142713-0ad062ec5ee5",
)

go_repository(
    name = "com_github_tommy_muehle_go_mnd",
    importpath = "github.com/tommy-muehle/go-mnd",
    sum = "h1:RC4maTWLKKwb7p1cnoygsbKIgNlJqSYBeAFON3Ar8As=",
    version = "v1.3.1-0.20200224220436-e6f9a994e8fa",
)

go_repository(
    name = "com_github_trivago_tgo",
    importpath = "github.com/trivago/tgo",
    sum = "h1:bxatjJIXNIpV18bucU4Uk/LaoxvxuOlp/oowRHyncLQ=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_tsenart_vegeta",
    importpath = "github.com/tsenart/vegeta",
    sum = "h1:ErZrHhRveAoznVW80gbrxz+qxJNydpA2fcQxTPHkZbU=",
    version = "v12.7.1-0.20190725001342-b5f4fca92137+incompatible",
)

go_repository(
    name = "com_github_ugorji_go",
    importpath = "github.com/ugorji/go",
    sum = "h1:j4s+tAvLfL3bZyefP2SEWmhBzmuIlH/eqNuPdFPgngw=",
    version = "v1.1.4",
)

go_repository(
    name = "com_github_ugorji_go_codec",
    importpath = "github.com/ugorji/go/codec",
    sum = "h1:3SVOIvH7Ae1KRYyQWRjXWJEA9sS/c/pjvH++55Gr648=",
    version = "v0.0.0-20181204163529-d75b2dcb6bc8",
)

go_repository(
    name = "com_github_ulikunitz_xz",
    importpath = "github.com/ulikunitz/xz",
    sum = "h1:YvTNdFzX6+W5m9msiYg/zpkSURPPtOlzbqYjrFn7Yt4=",
    version = "v0.5.7",
)

go_repository(
    name = "com_github_ultraware_funlen",
    importpath = "github.com/ultraware/funlen",
    sum = "h1:Av96YVBwwNSe4MLR7iI/BIa3VyI7/djnto/pK3Uxbdo=",
    version = "v0.0.2",
)

go_repository(
    name = "com_github_ultraware_whitespace",
    importpath = "github.com/ultraware/whitespace",
    sum = "h1:If7Va4cM03mpgrNH9k49/VOicWpGoG70XPBFFODYDsg=",
    version = "v0.0.4",
)

go_repository(
    name = "com_github_urfave_cli",
    importpath = "github.com/urfave/cli",
    sum = "h1:u7tSpNPPswAFymm8IehJhy4uJMlUuU/GmqSkvJ1InXA=",
    version = "v1.22.4",
)

go_repository(
    name = "com_github_urfave_cli_v2",
    importpath = "github.com/urfave/cli/v2",
    sum = "h1:Qt8FeAtxE/vfdrLmR3rxR6JRE0RoVmbXu8+6kZtYU4k=",
    version = "v2.1.1",
)

go_repository(
    name = "com_github_uudashr_gocognit",
    importpath = "github.com/uudashr/gocognit",
    sum = "h1:MoG2fZ0b/Eo7NXoIwCVFLG5JED3qgQz5/NEE+rOsjPs=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_valyala_bytebufferpool",
    importpath = "github.com/valyala/bytebufferpool",
    sum = "h1:GqA5TC/0021Y/b9FG4Oi9Mr3q7XYx6KllzawFIhcdPw=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_valyala_fasthttp",
    importpath = "github.com/valyala/fasthttp",
    sum = "h1:dzZJf2IuMiclVjdw0kkT+f9u4YdrapbNyGAN47E/qnk=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_valyala_quicktemplate",
    importpath = "github.com/valyala/quicktemplate",
    sum = "h1:BaO1nHTkspYzmAjPXj0QiDJxai96tlcZyKcI9dyEGvM=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_valyala_tcplisten",
    importpath = "github.com/valyala/tcplisten",
    sum = "h1:0R4NLDRDZX6JcmhJgXi5E4b8Wg84ihbmUKp/GvSPEzc=",
    version = "v0.0.0-20161114210144-ceec8f93295a",
)

go_repository(
    name = "com_github_vdemeester_k8s_pkg_credentialprovider",
    importpath = "github.com/vdemeester/k8s-pkg-credentialprovider",
    sum = "h1:czKEIG2Q3YRTgs6x/8xhjVMJD5byPo6cZuostkbTM74=",
    version = "v1.17.4",
)

go_repository(
    name = "com_github_vektah_gqlparser",
    importpath = "github.com/vektah/gqlparser",
    sum = "h1:ZsyLGn7/7jDNI+y4SEhI4yAxRChlv15pUHMjijT+e68=",
    version = "v1.1.2",
)

go_repository(
    name = "com_github_venafi_vcert_v4",
    importpath = "github.com/Venafi/vcert/v4",
    sum = "h1:37gfyjS9v5YvZcIABwNPo1fAC31lIZT7glVK1vfUxk4=",
    version = "v4.11.0",
)

go_repository(
    name = "com_github_vmware_govmomi",
    importpath = "github.com/vmware/govmomi",
    sum = "h1:gpw/0Ku+6RgF3jsi7fnCLmlcikBHfKBCUcu1qgc16OU=",
    version = "v0.20.3",
)

go_repository(
    name = "com_github_xanzy_go_gitlab",
    importpath = "github.com/xanzy/go-gitlab",
    sum = "h1:tBm+OXv1t+KBsqlXkSDFz+YUjRM0GFsjpOWYOod3Ebs=",
    version = "v0.32.0",
)

go_repository(
    name = "com_github_xanzy_ssh_agent",
    importpath = "github.com/xanzy/ssh-agent",
    sum = "h1:wUMzuKtKilRgBAD1sUb8gOwwRr2FGoBVumcjoOACClI=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_xdg_scram",
    importpath = "github.com/xdg/scram",
    sum = "h1:u40Z8hqBAAQyv+vATcGgV0YCnDjqSL7/q/JyPhhJSPk=",
    version = "v0.0.0-20180814205039-7eeb5667e42c",
)

go_repository(
    name = "com_github_xdg_stringprep",
    importpath = "github.com/xdg/stringprep",
    sum = "h1:d9X0esnoa3dFsV0FG35rAT0RIhYFlPq7MiP+DW89La0=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_xeipuuv_gojsonpointer",
    importpath = "github.com/xeipuuv/gojsonpointer",
    sum = "h1:J9EGpcZtP0E/raorCMxlFGSTBrsSlaDGf3jU/qvAE2c=",
    version = "v0.0.0-20180127040702-4e3ac2762d5f",
)

go_repository(
    name = "com_github_xeipuuv_gojsonreference",
    importpath = "github.com/xeipuuv/gojsonreference",
    sum = "h1:EzJWgHovont7NscjpAxXsDA8S8BMYve8Y5+7cuRE7R0=",
    version = "v0.0.0-20180127040603-bd5ef7bd5415",
)

go_repository(
    name = "com_github_xeipuuv_gojsonschema",
    importpath = "github.com/xeipuuv/gojsonschema",
    sum = "h1:ngVtJC9TY/lg0AA/1k48FYhBrhRoFlEmWzsehpNAaZg=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_xi2_xz",
    importpath = "github.com/xi2/xz",
    sum = "h1:nIPpBwaJSVYIxUFsDv3M8ofmx9yWTog9BfvIu0q41lo=",
    version = "v0.0.0-20171230120015-48954b6210f8",
)

go_repository(
    name = "com_github_xiang90_probing",
    importpath = "github.com/xiang90/probing",
    sum = "h1:eY9dn8+vbi4tKz5Qo6v2eYzo7kUS51QINcR5jNpbZS8=",
    version = "v0.0.0-20190116061207-43a291ad63a2",
)

go_repository(
    name = "com_github_xlab_handysort",
    importpath = "github.com/xlab/handysort",
    sum = "h1:j2hhcujLRHAg872RWAV5yaUrEjHEObwDv3aImCaNLek=",
    version = "v0.0.0-20150421192137-fb3537ed64a1",
)

go_repository(
    name = "com_github_xordataexchange_crypt",
    importpath = "github.com/xordataexchange/crypt",
    sum = "h1:ESFSdwYZvkeru3RtdrYueztKhOBCSAAzS4Gf+k0tEow=",
    version = "v0.0.3-0.20170626215501-b2862e3d0a77",
)

go_repository(
    name = "com_github_yuin_goldmark",
    importpath = "github.com/yuin/goldmark",
    sum = "h1:ruQGxdhGHe7FWOJPT0mKs5+pD2Xs1Bm/kdGlHO04FmM=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_yvasiyarov_go_metrics",
    importpath = "github.com/yvasiyarov/go-metrics",
    sum = "h1:+lm10QQTNSBd8DVTNGHx7o/IKu9HYDvLMffDhbyLccI=",
    version = "v0.0.0-20140926110328-57bccd1ccd43",
)

go_repository(
    name = "com_github_yvasiyarov_gorelic",
    importpath = "github.com/yvasiyarov/gorelic",
    sum = "h1:hlE8//ciYMztlGpl/VA+Zm1AcTPHYkHJPbHqE6WJUXE=",
    version = "v0.0.0-20141212073537-a9bba5b9ab50",
)

go_repository(
    name = "com_github_yvasiyarov_newrelic_platform_go",
    importpath = "github.com/yvasiyarov/newrelic_platform_go",
    sum = "h1:ERexzlUfuTvpE74urLSbIQW0Z/6hF9t8U4NsJLaioAY=",
    version = "v0.0.0-20140908184405-b21fdbd4370f",
)

go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    sum = "h1:DZeAkuQGQqnm9Xv36SbMJEU8aFBz4wL04UpMWPWwjzg=",
    version = "v0.66.0",
)

go_repository(
    name = "com_google_cloud_go_bigquery",
    importpath = "cloud.google.com/go/bigquery",
    sum = "h1:PQcPefKFdaIzjQFbiyOgAqyx8q5djaE7x9Sqe712DPA=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_datastore",
    importpath = "cloud.google.com/go/datastore",
    sum = "h1:/May9ojXjRkPBNVrq+oWLqmWCkr4OU5uRY29bu0mRyQ=",
    version = "v1.1.0",
)

go_repository(
    name = "com_google_cloud_go_firestore",
    importpath = "cloud.google.com/go/firestore",
    sum = "h1:9x7Bx0A9R5/M9jibeJeZWqjeVEIxYW9fZYqB9a70/bY=",
    version = "v1.1.0",
)

go_repository(
    name = "com_google_cloud_go_logging",
    importpath = "cloud.google.com/go/logging",
    sum = "h1:kaunpnoEh9L4hu6JUsBa8Y20LBfKnCuDhKUgdZp7oK8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_google_cloud_go_pubsub",
    importpath = "cloud.google.com/go/pubsub",
    sum = "h1:76oR7VBOkL7ivoIrFKyW0k7YDCRelrlxktIzQiIUGgg=",
    version = "v1.4.0",
)

go_repository(
    name = "com_google_cloud_go_storage",
    importpath = "cloud.google.com/go/storage",
    sum = "h1:4y3gHptW1EHVtcPAVE0eBBlFuGqEejTTG3KdIE0lUX4=",
    version = "v1.12.0",
)

go_repository(
    name = "com_shuralyov_dmitri_gpu_mtl",
    importpath = "dmitri.shuralyov.com/gpu/mtl",
    sum = "h1:VpgP7xuJadIUuKccphEpTJnWhS2jkQyMt6Y7pJCD7fY=",
    version = "v0.0.0-20190408044501-666a987793e9",
)

go_repository(
    name = "com_sourcegraph_sqs_pbtypes",
    importpath = "sourcegraph.com/sqs/pbtypes",
    sum = "h1:f7lAwqviDEGvON4kRv0o5V7FT/IQK+tbkF664XMbP3o=",
    version = "v1.0.0",
)

go_repository(
    name = "com_sslmate_software_src_go_pkcs12",
    importpath = "software.sslmate.com/src/go-pkcs12",
    sum = "h1:AVd6O+azYjVQYW1l55IqkbL8/JxjrLtO6q4FCmV8N5c=",
    version = "v0.0.0-20200830195227-52f69702a001",
)

go_repository(
    name = "dev_gocloud",
    importpath = "gocloud.dev",
    sum = "h1:EDRyaRAnMGSq/QBto486gWFxMLczAfIYUmusV7XLNBM=",
    version = "v0.19.0",
)

go_repository(
    name = "dev_knative_caching",
    importpath = "knative.dev/caching",
    sum = "h1:mxrur6DsVK8uIjhIq7c1OMls4YjBcRlyvnh3Vx13a0M=",
    version = "v0.0.0-20200116200605-67bca2c83dfa",
)

go_repository(
    name = "dev_knative_eventing_contrib",
    importpath = "knative.dev/eventing-contrib",
    sum = "h1:xncT+JrokPG+hPUFJwue8ubPpzmziV9GUIZqYt01JDo=",
    version = "v0.11.2",
)

go_repository(
    name = "dev_knative_pkg",
    importpath = "knative.dev/pkg",
    sum = "h1:NDQS+236vhwCP9oiBBGvQ5WGzbD0Y8Pcv9dtE2stg+Q=",
    version = "v0.0.0-20200711004937-22502028e31a",
)

go_repository(
    name = "dev_knative_test_infra",
    importpath = "knative.dev/test-infra",
    sum = "h1:wNlGK4f5Ykqh3KLC5RlyR9kvzvRgo/LwJQNsZWGVHnU=",
    version = "v0.0.0-20200707183444-aed09e56ddc7",
)

go_repository(
    name = "in_gopkg_airbrake_gobrake_v2",
    importpath = "gopkg.in/airbrake/gobrake.v2",
    sum = "h1:7z2uVWwn7oVeeugY1DtlPAy5H+KYgB1KeKTnqjNatLo=",
    version = "v2.0.9",
)

go_repository(
    name = "in_gopkg_alecthomas_kingpin_v2",
    importpath = "gopkg.in/alecthomas/kingpin.v2",
    sum = "h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=",
    version = "v2.2.6",
)

go_repository(
    name = "in_gopkg_asn1_ber_v1",
    importpath = "gopkg.in/asn1-ber.v1",
    sum = "h1:TxyelI5cVkbREznMhfzycHdkp5cLA7DpE+GKjSslYhM=",
    version = "v1.0.0-20181015200546-f715ec2f112d",
)

go_repository(
    name = "in_gopkg_check_v1",
    importpath = "gopkg.in/check.v1",
    sum = "h1:Hei/4ADfdWqJk1ZMxUNpqntNwaWcugrBjAiHlqqRiVk=",
    version = "v1.0.0-20201130134442-10cb98267c6c",
)

go_repository(
    name = "in_gopkg_cheggaaa_pb_v1",
    importpath = "gopkg.in/cheggaaa/pb.v1",
    sum = "h1:Ev7yu1/f6+d+b3pi5vPdRPc6nNtP1umSfcWiEfRqv6I=",
    version = "v1.0.25",
)

go_repository(
    name = "in_gopkg_errgo_v2",
    importpath = "gopkg.in/errgo.v2",
    sum = "h1:0vLT13EuvQ0hNvakwLuFZ/jYrLp5F3kcWHXdRggjCE8=",
    version = "v2.1.0",
)

go_repository(
    name = "in_gopkg_fsnotify_v1",
    importpath = "gopkg.in/fsnotify.v1",
    sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
    version = "v1.4.7",
)

go_repository(
    name = "in_gopkg_gcfg_v1",
    importpath = "gopkg.in/gcfg.v1",
    sum = "h1:0HIbH907iBTAntm+88IJV2qmJALDAh8sPekI9Vc1fm0=",
    version = "v1.2.0",
)

go_repository(
    name = "in_gopkg_gemnasium_logrus_airbrake_hook_v2",
    importpath = "gopkg.in/gemnasium/logrus-airbrake-hook.v2",
    sum = "h1:OAj3g0cR6Dx/R07QgQe8wkA9RNjB2u4i700xBkIT4e0=",
    version = "v2.1.2",
)

go_repository(
    name = "in_gopkg_inf_v0",
    importpath = "gopkg.in/inf.v0",
    sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
    version = "v0.9.1",
)

go_repository(
    name = "in_gopkg_ini_v1",
    importpath = "gopkg.in/ini.v1",
    sum = "h1:DPMeDvGTM54DXbPkVIZsp19fp/I2K7zwA/itHYHKo8Y=",
    version = "v1.56.0",
)

go_repository(
    name = "in_gopkg_jcmturner_aescts_v1",
    importpath = "gopkg.in/jcmturner/aescts.v1",
    sum = "h1:cVVZBK2b1zY26haWB4vbBiZrfFQnfbTVrE3xZq6hrEw=",
    version = "v1.0.1",
)

go_repository(
    name = "in_gopkg_jcmturner_dnsutils_v1",
    importpath = "gopkg.in/jcmturner/dnsutils.v1",
    sum = "h1:cIuC1OLRGZrld+16ZJvvZxVJeKPsvd5eUIvxfoN5hSM=",
    version = "v1.0.1",
)

go_repository(
    name = "in_gopkg_jcmturner_gokrb5_v7",
    importpath = "gopkg.in/jcmturner/gokrb5.v7",
    sum = "h1:0709Jtq/6QXEuWRfAm260XqlpcwL1vxtO1tUE2qK8Z4=",
    version = "v7.3.0",
)

go_repository(
    name = "in_gopkg_jcmturner_rpc_v1",
    importpath = "gopkg.in/jcmturner/rpc.v1",
    sum = "h1:QHIUxTX1ISuAv9dD2wJ9HWQVuWDX/Zc0PfeC2tjc4rU=",
    version = "v1.1.0",
)

go_repository(
    name = "in_gopkg_natefinch_lumberjack_v2",
    importpath = "gopkg.in/natefinch/lumberjack.v2",
    sum = "h1:1Lc07Kr7qY4U2YPouBjpCLxpiyxIVoxqXgkXLknAOE8=",
    version = "v2.0.0",
)

go_repository(
    name = "in_gopkg_resty_v1",
    importpath = "gopkg.in/resty.v1",
    sum = "h1:CuXP0Pjfw9rOuY6EP+UvtNvt5DSqHpIxILZKT/quCZI=",
    version = "v1.12.0",
)

go_repository(
    name = "in_gopkg_robfig_cron_v2",
    importpath = "gopkg.in/robfig/cron.v2",
    sum = "h1:E846t8CnR+lv5nE+VuiKTDG/v1U2stad0QzddfJC7kY=",
    version = "v2.0.0-20150107220207-be2e0b0deed5",
)

go_repository(
    name = "in_gopkg_square_go_jose_v2",
    importpath = "gopkg.in/square/go-jose.v2",
    sum = "h1:SK5KegNXmKmqE342YYN2qPHEnUYeoMiXXl1poUlI+o4=",
    version = "v2.3.1",
)

go_repository(
    name = "in_gopkg_src_d_go_billy_v4",
    importpath = "gopkg.in/src-d/go-billy.v4",
    sum = "h1:0SQA1pRztfTFx2miS8sA97XvooFeNOmvUenF4o0EcVg=",
    version = "v4.3.2",
)

go_repository(
    name = "in_gopkg_src_d_go_git_fixtures_v3",
    importpath = "gopkg.in/src-d/go-git-fixtures.v3",
    sum = "h1:ivZFOIltbce2Mo8IjzUHAFoq/IylO9WHhNOAJK+LsJg=",
    version = "v3.5.0",
)

go_repository(
    name = "in_gopkg_src_d_go_git_v4",
    importpath = "gopkg.in/src-d/go-git.v4",
    sum = "h1:SRtFyV8Kxc0UP7aCHcijOMQGPxHSmMOPrzulQWolkYE=",
    version = "v4.13.1",
)

go_repository(
    name = "in_gopkg_tomb_v1",
    importpath = "gopkg.in/tomb.v1",
    sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
    version = "v1.0.0-20141024135613-dd632973f1e7",
)

go_repository(
    name = "in_gopkg_warnings_v0",
    importpath = "gopkg.in/warnings.v0",
    sum = "h1:wFXVbFY8DY5/xOe1ECiWdKCzZlxgshcYVNkBHstARME=",
    version = "v0.1.2",
)

go_repository(
    name = "in_gopkg_yaml_v1",
    importpath = "gopkg.in/yaml.v1",
    sum = "h1:POO/ycCATvegFmVuPpQzZFJ+pGZeX22Ufu6fibxDVjU=",
    version = "v1.0.0-20140924161607-9f9df34309c0",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    sum = "h1:clyUAQHOM3G0M3f5vQj7LuJrETvjVot3Z5el9nffUtU=",
    version = "v2.3.0",
)

go_repository(
    name = "in_gopkg_yaml_v3",
    importpath = "gopkg.in/yaml.v3",
    sum = "h1:tQIYjPdBoyREyB9XMu+nnTclpTYkz2zFM+lzLJFO4gQ=",
    version = "v3.0.0-20200615113413-eeeca48fe776",
)

go_repository(
    name = "io_etcd_go_bbolt",
    importpath = "go.etcd.io/bbolt",
    sum = "h1:XAzx9gjCb0Rxj7EoqcClPD1d5ZBxZJk0jbuoPHenBt0=",
    version = "v1.3.5",
)

go_repository(
    name = "io_etcd_go_etcd",
    importpath = "go.etcd.io/etcd",
    sum = "h1:Gqga3zA9tdAcfqobUGjSoCob5L3f8Dt5EuOp3ihNZko=",
    version = "v0.5.0-alpha.5.0.20200819165624-17cef6e3e9d5",
)

go_repository(
    name = "io_gitea_code_sdk_gitea",
    importpath = "code.gitea.io/sdk/gitea",
    sum = "h1:hvDCz4wtFvo7rf5Ebj8tGd4aJ4wLPKX3BKFX9Dk1Pgs=",
    version = "v0.12.0",
)

go_repository(
    name = "io_k8s_api",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/api",
    replace = "k8s.io/api",
    sum = "h1:GN6ntFnv44Vptj/b+OnMW7FmzkpDoIDLZRvKX3XH9aU=",
    version = "v0.19.3",
)

go_repository(
    name = "io_k8s_apiextensions_apiserver",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/apiextensions-apiserver",
    sum = "h1:oG84UwiDsVDu7dlsGQs5GySmQHCzMhknfhFExJMz9tA=",
    version = "v0.19.2",
)

go_repository(
    name = "io_k8s_apimachinery",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/apimachinery",
    replace = "k8s.io/apimachinery",
    sum = "h1:bpIQXlKjB4cB/oNpnNnV+BybGPR7iP5oYpsOTEJ4hgc=",
    version = "v0.19.3",
)

go_repository(
    name = "io_k8s_apiserver",
    importpath = "k8s.io/apiserver",
    sum = "h1:xq2dXAzsAoHv7S4Xc/p7PKhiowdHV/PgdePWo3MxIYM=",
    version = "v0.19.2",
)

go_repository(
    name = "io_k8s_cli_runtime",
    importpath = "k8s.io/cli-runtime",
    sum = "h1:wLe+osHSqcItyS3MYQXVyGFa54fppORVA8Jn7DBGSWw=",
    version = "v0.19.0",
)

go_repository(
    name = "io_k8s_client_go",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/client-go",
    replace = "k8s.io/client-go",
    sum = "h1:ctqR1nQ52NUs6LpI0w+a5U+xjYwflFwA13OJKcicMxg=",
    version = "v0.19.3",
)

go_repository(
    name = "io_k8s_cloud_provider",
    importpath = "k8s.io/cloud-provider",
    sum = "h1:ELMIQwweSNu8gfVEnLDypxd9034S1sZJg6QcdWJOvMI=",
    version = "v0.17.4",
)

go_repository(
    name = "io_k8s_code_generator",
    importpath = "k8s.io/code-generator",
    replace = "k8s.io/code-generator",
    sum = "h1:fTrTpJ8PZog5oo6MmeZtveo89emjQZHiw0ieybz1RSs=",
    version = "v0.19.3",
)

go_repository(
    name = "io_k8s_component_base",
    importpath = "k8s.io/component-base",
    sum = "h1:jW5Y9RcZTb79liEhW3XDVTW7MuvEGP0tQZnfSX6/+gs=",
    version = "v0.19.2",
)

go_repository(
    name = "io_k8s_csi_translation_lib",
    importpath = "k8s.io/csi-translation-lib",
    sum = "h1:bP9yGfCJDknP7tklCwizZtwgJNRePMVcEaFIfeA11ho=",
    version = "v0.17.4",
)

go_repository(
    name = "io_k8s_gengo",
    importpath = "k8s.io/gengo",
    sum = "h1:t4L10Qfx/p7ASH3gXCdIUtPbbIuegCoUJf3TMSFekjw=",
    version = "v0.0.0-20200428234225-8167cfdcfc14",
)

go_repository(
    name = "io_k8s_klog",
    importpath = "k8s.io/klog",
    sum = "h1:Pt+yjF5aB1xDSVbau4VsWe+dQNzA0qv1LlXdC2dF6Q8=",
    version = "v1.0.0",
)

go_repository(
    name = "io_k8s_klog_v2",
    importpath = "k8s.io/klog/v2",
    sum = "h1:WmkrnW7fdrm0/DMClc+HIxtftvxVIPAhlVwMQo5yLco=",
    version = "v2.3.0",
)

go_repository(
    name = "io_k8s_kube_aggregator",
    importpath = "k8s.io/kube-aggregator",
    sum = "h1:rL4fsftMaqkKjaibArYDaBeqN41CHaJzgRJjUB9IrIg=",
    version = "v0.19.0",
)

go_repository(
    name = "io_k8s_kube_openapi",
    importpath = "k8s.io/kube-openapi",
    sum = "h1:+WnxoVtG8TMiudHBSEtrVL1egv36TkkJm+bA8AxicmQ=",
    version = "v0.0.0-20200805222855-6aeccd4b50c6",
)

go_repository(
    name = "io_k8s_kubectl",
    importpath = "k8s.io/kubectl",
    sum = "h1:t9uxaZzGvqc2jY96mjnPSjFHtaKOxoUegeGZdaGT6aw=",
    version = "v0.19.0",
)

go_repository(
    name = "io_k8s_kubernetes",
    importpath = "k8s.io/kubernetes",
    sum = "h1:wJx/r2HuPVaaBeCUk/P47GSK0eyrj3mI/kESRFBp6/A=",
    version = "v1.14.7",
)

go_repository(
    name = "io_k8s_legacy_cloud_providers",
    importpath = "k8s.io/legacy-cloud-providers",
    sum = "h1:VvFqJGiYAr2gIdoNuqbeZLEdxIFeN4Yt6OLJS9l2oIE=",
    version = "v0.17.4",
)

go_repository(
    name = "io_k8s_metrics",
    importpath = "k8s.io/metrics",
    sum = "h1:cKq0+Z7wg5qkK1n8dryNffKfU22DBX83JguGpR+TCk0=",
    version = "v0.19.0",
)

go_repository(
    name = "io_k8s_sigs_apiserver_network_proxy_konnectivity_client",
    importpath = "sigs.k8s.io/apiserver-network-proxy/konnectivity-client",
    sum = "h1:rusRLrDhjBp6aYtl9sGEvQJr6faoHoDLd0YcUBTZguI=",
    version = "v0.0.9",
)

go_repository(
    name = "io_k8s_sigs_boskos",
    importpath = "sigs.k8s.io/boskos",
    sum = "h1:ofZy0Txrpd7wjp0cA/Ok5aVTYU1n2uu88hWrvRjUd3U=",
    version = "v0.0.0-20201218211225-4c6cb9eeb307",
)

go_repository(
    name = "io_k8s_sigs_controller_runtime",
    importpath = "sigs.k8s.io/controller-runtime",
    sum = "h1:Agy2ieyq35JZftrmJuyFZNlE/y/+xMYuHCbsliXhDEU=",
    version = "v0.7.0-alpha.6.0.20201106193838-8d0107636985",
)

go_repository(
    name = "io_k8s_sigs_controller_tools",
    importpath = "sigs.k8s.io/controller-tools",
    sum = "h1:PXOHvyYAjWfO0UfQvaUo33HpXNCOilV3i/Vjc7iM1/A=",
    version = "v0.2.9-0.20200414181213-645d44dca7c0",
)

go_repository(
    name = "io_k8s_sigs_kustomize",
    importpath = "sigs.k8s.io/kustomize",
    sum = "h1:JUufWFNlI44MdtnjUqVnvh29rR37PQFzPbLXqhyOyX0=",
    version = "v2.0.3+incompatible",
)

go_repository(
    name = "io_k8s_sigs_structured_merge_diff",
    importpath = "sigs.k8s.io/structured-merge-diff",
    sum = "h1:LOs1LZWMsz1xs77Phr/pkB4LFaavH7IVq/3+WTN9XTA=",
    version = "v1.0.1",
)

go_repository(
    name = "io_k8s_sigs_structured_merge_diff_v2",
    importpath = "sigs.k8s.io/structured-merge-diff/v2",
    sum = "h1:I0h4buiCqDtPztO3NOiyoNMtqSIfld49D4Wj3UBXYZA=",
    version = "v2.0.1",
)

go_repository(
    name = "io_k8s_sigs_structured_merge_diff_v3",
    importpath = "sigs.k8s.io/structured-merge-diff/v3",
    sum = "h1:dOmIZBMfhcHS09XZkMyUgkq5trg3/jRyJYFZUiaOp8E=",
    version = "v3.0.0",
)

go_repository(
    name = "io_k8s_sigs_structured_merge_diff_v4",
    importpath = "sigs.k8s.io/structured-merge-diff/v4",
    sum = "h1:YXTMot5Qz/X1iBRJhAt+vI+HVttY0WkSqqhKxQ0xVbA=",
    version = "v4.0.1",
)

go_repository(
    name = "io_k8s_sigs_testing_frameworks",
    importpath = "sigs.k8s.io/testing_frameworks",
    sum = "h1:vK0+tvjF0BZ/RYFeZ1E6BYBwHJJXhjuZ3TdsEKH+UQM=",
    version = "v0.1.2",
)

go_repository(
    name = "io_k8s_sigs_yaml",
    importpath = "sigs.k8s.io/yaml",
    sum = "h1:kr/MCeFWJWTwyaHoR9c8EjH9OumOmoF9YGiZd7lFm/Q=",
    version = "v1.2.0",
)

go_repository(
    name = "io_k8s_test_infra",
    importpath = "k8s.io/test-infra",
    sum = "h1:VR/F7vFfoOIHvMHZY04FOYhBEhb+syxY5U+b7KaG1/o=",
    version = "v0.0.0-20201214190528-57362ae63e51",
)

go_repository(
    name = "io_k8s_utils",
    importpath = "k8s.io/utils",
    sum = "h1:9ZNvfPvVIEsp/T1ez4GQuzCcCTEQWhovSofhqR73A6g=",
    version = "v0.0.0-20200912215256-4140de9c8800",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    sum = "h1:LYy1Hy3MJdrCdMwwzxA/dRok4ejH+RwNGbuoD9fCjto=",
    version = "v0.22.4",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_aws",
    importpath = "contrib.go.opencensus.io/exporter/aws",
    sum = "h1:YsbWYxDZkC7x2OxlsDEYvvEXZ3cBI3qBgUK5BqkZvRw=",
    version = "v0.0.0-20181029163544-2befc13012d0",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_ocagent",
    importpath = "contrib.go.opencensus.io/exporter/ocagent",
    sum = "h1:Z1n6UAyr0QwM284yUuh5Zd8JlvxUGAhFZcgMJkMPrGM=",
    version = "v0.6.0",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_prometheus",
    importpath = "contrib.go.opencensus.io/exporter/prometheus",
    sum = "h1:SByaIoWwNgMdPSgl5sMqM2KDE5H/ukPWBRo314xiDvg=",
    version = "v0.1.0",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_stackdriver",
    importpath = "contrib.go.opencensus.io/exporter/stackdriver",
    sum = "h1:RX9W6FelAqTVnBi/bRXJLXr9n18v4QkQwZYIdnNS51I=",
    version = "v0.13.1",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_zipkin",
    importpath = "contrib.go.opencensus.io/exporter/zipkin",
    sum = "h1:PR+1zWqY8ceXs1qDQQIlgXe+sdiwCf0n32bH4+Epk8g=",
    version = "v0.1.1",
)

go_repository(
    name = "io_opencensus_go_contrib_integrations_ocsql",
    importpath = "contrib.go.opencensus.io/integrations/ocsql",
    sum = "h1:kfg5Yyy1nYUrqzyfW5XX+dzMASky8IJXhtHe0KTYNS4=",
    version = "v0.1.4",
)

go_repository(
    name = "io_opencensus_go_contrib_resource",
    importpath = "contrib.go.opencensus.io/resource",
    sum = "h1:4r2CANuYhKGmYWP02+5E94rLRcS/YeD+KlxSrOsMxk0=",
    version = "v0.1.1",
)

go_repository(
    name = "io_rsc_binaryregexp",
    importpath = "rsc.io/binaryregexp",
    sum = "h1:HfqmD5MEmC0zvwBuF187nq9mdnXjXsSivRiXN7SmRkE=",
    version = "v0.2.0",
)

go_repository(
    name = "io_rsc_letsencrypt",
    importpath = "rsc.io/letsencrypt",
    sum = "h1:H7xDfhkaFFSYEJlKeq38RwX2jYcnTeHuDQyT+mMNMwM=",
    version = "v0.0.3",
)

go_repository(
    name = "io_rsc_quote_v3",
    importpath = "rsc.io/quote/v3",
    sum = "h1:9JKUTTIUgS6kzR9mK1YuGKv6Nl+DijDNIc0ghT58FaY=",
    version = "v3.1.0",
)

go_repository(
    name = "io_rsc_sampler",
    importpath = "rsc.io/sampler",
    sum = "h1:7uVkIFmeBqHfdjD+gZwtXXI+RODJ2Wc4O7MPEh/QiW4=",
    version = "v1.3.0",
)

go_repository(
    name = "ml_vbom_util",
    importpath = "vbom.ml/util",
    sum = "h1:O69FD9pJA4WUZlEwYatBEEkRWKQ5cKodWpdKTrCS/iQ=",
    version = "v0.0.0-20180919145318-efcd4e0f9787",
)

go_repository(
    name = "org_apache_git_thrift_git",
    importpath = "git.apache.org/thrift.git",
    sum = "h1:CMxsZlAmxKs+VAZMlDDL0wXciMblJcutQbEe3A9CYUM=",
    version = "v0.12.0",
)

go_repository(
    name = "org_bazil_fuse",
    importpath = "bazil.org/fuse",
    sum = "h1:FNCRpXiquG1aoyqcIWVFmpTSKVcx2bQD38uZZeGtdlw=",
    version = "v0.0.0-20180421153158-65cc252bf669",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    sum = "h1:Le77IccnTqEa8ryp9wIpX5W3zYm7Gf9LhOp9PHcwFts=",
    version = "v0.32.0",
)

go_repository(
    name = "org_golang_google_appengine",
    importpath = "google.golang.org/appengine",
    sum = "h1:lMO5rYAqUxkmaj76jAkRUvt5JZgFymx/+Q5Mzfivuhc=",
    version = "v1.6.6",
)

go_repository(
    name = "org_golang_google_cloud",
    importpath = "google.golang.org/cloud",
    sum = "h1:Cpp2P6TPjujNoC5M2KHY6g7wfyLYfIWRZaSdIKfDasA=",
    version = "v0.0.0-20151119220103-975617b05ea8",
)

go_repository(
    name = "org_golang_google_genproto",
    importpath = "google.golang.org/genproto",
    sum = "h1:B9nroC8SSX5GtbVvxPF9tYIVkaCpjhVLOrlAY8ONzm8=",
    version = "v0.0.0-20200921151605-7abf4a1a14d5",
)

go_repository(
    name = "org_golang_google_protobuf",
    importpath = "google.golang.org/protobuf",
    sum = "h1:Ejskq+SyPohKW+1uil0JJMtmHCgJPJ/qWTxr8qp+R4c=",
    version = "v1.25.0",
)

go_repository(
    name = "org_golang_x_crypto",
    importpath = "golang.org/x/crypto",
    sum = "h1:It14KIkyBFYkHkwZ7k45minvA9aorojkyjGk9KJ5B/w=",
    version = "v0.0.0-20210322153248-0c34fe9e7dc2",
)

go_repository(
    name = "org_golang_x_exp",
    importpath = "golang.org/x/exp",
    sum = "h1:QE6XYQK6naiK1EPAe1g/ILLxN5RBoH5xkJk3CqlMI/Y=",
    version = "v0.0.0-20200224162631-6cc2880d07d6",
)

go_repository(
    name = "org_golang_x_image",
    importpath = "golang.org/x/image",
    sum = "h1:+qEpEAPhDZ1o0x3tHzZTQDArnOixOzGD9HUJfcg0mb4=",
    version = "v0.0.0-20190802002840-cff245a6509b",
)

go_repository(
    name = "org_golang_x_lint",
    importpath = "golang.org/x/lint",
    replace = "golang.org/x/lint",
    sum = "h1:QzoH/1pFpZguR8NrRHLcO6jKqfv2zpuSqZLgdm7ZmjI=",
    version = "v0.0.0-20190409202823-959b441ac422",
)

go_repository(
    name = "org_golang_x_mobile",
    importpath = "golang.org/x/mobile",
    sum = "h1:b373EGXtj0o+ssqkOkdVphTCZ/fVg2LwhctJn2QQbqA=",
    version = "v0.0.0-20190806162312-597adff16ade",
)

go_repository(
    name = "org_golang_x_mod",
    importpath = "golang.org/x/mod",
    sum = "h1:8pl+sMODzuvGJkmj2W4kZihvVb5mKm8pB/X44PIQHv8=",
    version = "v0.4.0",
)

go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    sum = "h1:ld7aEMNHoBnnDAX15v1T6z31v8HwR2A9FYOuAhWqkwc=",
    version = "v0.0.0-20200902213428-5d25da1a8d43",
)

go_repository(
    name = "org_golang_x_sync",
    importpath = "golang.org/x/sync",
    sum = "h1:DcqTD9SDLc+1P/r1EmRBwnVsrOwW+kk2vWf9n+1sGhs=",
    version = "v0.0.0-20201207232520-09787c993a3a",
)

go_repository(
    name = "org_golang_x_sys",
    importpath = "golang.org/x/sys",
    sum = "h1:Paq34FxTluEPvVyayQqMPgHm+vTOrIifmcYxFBx9TLg=",
    version = "v0.0.0-20210324051608-47abb6519492",
)

go_repository(
    name = "org_golang_x_time",
    importpath = "golang.org/x/time",
    sum = "h1:EHBhcS0mlXEAVwNyO2dLfjToGsyY4j24pTs2ScHnX7s=",
    version = "v0.0.0-20200630173020-3af7569d3a1e",
)

go_repository(
    name = "org_golang_x_tools",
    importpath = "golang.org/x/tools",
    sum = "h1:xx3+TTG3yS1I6Ola5Kapxr5vZu85vKkcwKyV6ke9fHA=",
    version = "v0.0.0-20201230224404-63754364767c",
)

go_repository(
    name = "org_golang_x_xerrors",
    importpath = "golang.org/x/xerrors",
    sum = "h1:go1bK/D/BFZV2I8cIQd1NKEZ+0owSTG1fDTci4IqFcE=",
    version = "v0.0.0-20200804184101-5ec99f83aff1",
)

go_repository(
    name = "org_gonum_v1_gonum",
    importpath = "gonum.org/v1/gonum",
    sum = "h1:PupagGYwj8+I4ubCxcmcBRk3VlUWtTg5huQpZR9flmE=",
    version = "v0.0.0-20181121035319-3f7ecaa7e8ca",
)

go_repository(
    name = "org_gonum_v1_netlib",
    importpath = "gonum.org/v1/netlib",
    sum = "h1:4WsZyVtkthqrHTbDCJfiTs8IWNYE4uvsSDgaV6xpp+o=",
    version = "v0.0.0-20181029234149-ec6d1f5cefe6",
)

go_repository(
    name = "org_mongodb_go_mongo_driver",
    importpath = "go.mongodb.org/mongo-driver",
    sum = "h1:jxcFYjlkl8xaERsgLo+RNquI0epW6zuy/ZRQs6jnrFA=",
    version = "v1.1.2",
)

go_repository(
    name = "org_uber_go_atomic",
    importpath = "go.uber.org/atomic",
    sum = "h1:Ezj3JGmsOnG1MoRWQkPBsKLe9DwWD9QeXzTRzzldNVk=",
    version = "v1.6.0",
)

go_repository(
    name = "org_uber_go_goleak",
    importpath = "go.uber.org/goleak",
    sum = "h1:z+mqJhf6ss6BSfSM671tgKyZBFPTTJM+HLxnhPC3wu0=",
    version = "v1.1.10",
)

go_repository(
    name = "org_uber_go_multierr",
    importpath = "go.uber.org/multierr",
    sum = "h1:KCa4XfM8CWFCpxXRGok+Q0SS/0XBhMDbHHGABQLvD2A=",
    version = "v1.5.0",
)

go_repository(
    name = "org_uber_go_tools",
    importpath = "go.uber.org/tools",
    sum = "h1:0mgffUl7nfd+FpvXMVz4IDEaUSmT1ysygQC7qYo7sG4=",
    version = "v0.0.0-20190618225709-2cfd321de3ee",
)

go_repository(
    name = "org_uber_go_zap",
    importpath = "go.uber.org/zap",
    sum = "h1:ZZCA22JRF2gQE5FoNmhmrf7jeJJ2uhqDUNRYKm8dvmM=",
    version = "v1.15.0",
)

go_repository(
    name = "sh_helm_helm_v3",
    importpath = "helm.sh/helm/v3",
    sum = "h1:aykwPMVyQyncZ8iLNVMXgJ1l3c6W0+LSOPmqp8JdCjs=",
    version = "v3.1.1",
)

go_repository(
    name = "tools_gotest",
    importpath = "gotest.tools",
    sum = "h1:VsBPFP1AI068pPrMxtb/S8Zkgf9xEmTLJjfM+P5UIEo=",
    version = "v2.2.0+incompatible",
)

go_repository(
    name = "tools_gotest_v3",
    importpath = "gotest.tools/v3",
    sum = "h1:kG1BFyqVHuQoVQiR1bWGnfz/fmHvvuiSPIV7rvl360E=",
    version = "v3.0.2",
)

go_repository(
    name = "xyz_gomodules_jsonpatch_v2",
    importpath = "gomodules.xyz/jsonpatch/v2",
    sum = "h1:Phva6wqu+xR//Njw6iorylFFgn/z547tw5Ne3HZPQ+k=",
    version = "v2.1.0",
)

go_repository(
    name = "com_github_go_git_gcfg",
    importpath = "github.com/go-git/gcfg",
    sum = "h1:Q5ViNfGF8zFgyJWPqYwA7qGFoMTEiBmdlkcfRmpIMa4=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_go_git_go_billy_v5",
    importpath = "github.com/go-git/go-billy/v5",
    sum = "h1:4pl5BV4o7ZG/lterP4S6WzJ6xr49Ba5ET9ygheTYahk=",
    version = "v5.1.0",
)

go_repository(
    name = "com_github_go_git_go_git_fixtures_v4",
    importpath = "github.com/go-git/go-git-fixtures/v4",
    sum = "h1:PbKy9zOy4aAKrJ5pibIRpVO2BXnK1Tlcg+caKI7Ox5M=",
    version = "v4.0.2-0.20200613231340-f56387b50c12",
)

go_repository(
    name = "com_github_go_git_go_git_v5",
    importpath = "github.com/go-git/go-git/v5",
    sum = "h1:8WKMtJR2j8RntEXR/uvTKagfEt4GYlwQ7mntE4+0GWc=",
    version = "v5.3.0",
)

go_repository(
    name = "org_golang_x_term",
    importpath = "golang.org/x/term",
    sum = "h1:v+OssWQX+hTHEmOBgwxdZxK4zHq3yOs8F9J7mk0PY8E=",
    version = "v0.0.0-20201126162022-7de9c90e9dd1",
)
