load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "rules_python",
    sha256 = "934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
    url = "https://github.com/bazelbuild/rules_python/releases/download/0.3.0/rules_python-0.3.0.tar.gz",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "b1e80761a8a8243d03ebca8845e9cc1ba6c82ce7c5179ce2b295cd36f7e394bf",
    #strip_prefix = "rules_docker-0.25.0",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.25.0/rules_docker-v0.25.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    integrity = "sha256-fHbWI2so/2laoozzX5XeMXqUcv0fsUrHl8m/aE8Js3w=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.44.2/rules_go-v0.44.2.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.44.2/rules_go-v0.44.2.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    integrity = "sha256-MpOL2hbmcABjA1R5Bj2dJMYO2o15/Uc5Vj9Q0zHLMgk=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
    ],
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "d0f5f605d0d656007ce6c8b5a82df3037e1d8fe8b121ed42e536f569dec16113",
    strip_prefix = "protobuf-3.14.0",
    urls = [
        "https://mirror.bazel.build/github.com/protocolbuffers/protobuf/archive/v3.14.0.tar.gz",
        "https://github.com/protocolbuffers/protobuf/archive/v3.14.0.tar.gz",
    ],
)

load("//:go_third_party.bzl", "go_deps")

# gazelle:repository_macro go_third_party.bzl%go_deps
go_deps()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "1.21.6")

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
    repository = "k8s-staging-test-infra/bootstrap",
    tag = "v20240110-cc03552828",
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

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

go_repository(
    name = "com_github_bazelbuild_buildtools",
    importpath = "github.com/bazelbuild/buildtools",
    sum = "h1:OhVnC5zU5QHQ+DUSmgOTPqPnJnrlFmrh2S0HKeHmpbw=",
    version = "v0.0.0-20200922170545-10384511ce98",
)

go_repository(
    name = "com_github_bndr_gojenkins",
    importpath = "github.com/bndr/gojenkins",
    sum = "h1:TWyJI6ST1qDAfH33DQb3G4mD8KkrBfyfSUoZBHQAvPI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_avast_retry_go",
    importpath = "github.com/avast/retry-go",
    sum = "h1:4SOWQ7Qs+oroOTQOYnAHqelpCO0biHSxpiH9JdtuBj0=",
    version = "v3.0.0+incompatible",
)

go_repository(
    name = "com_github_lnquy_cron",
    importpath = "github.com/lnquy/cron",
    sum = "h1:iaDX1ublgQ9LBhA8l9BVU+FrTE1PPSPAuvAdhgdnXgA=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_ajstarks_deck",
    importpath = "github.com/ajstarks/deck",
    sum = "h1:7kQgkwGRoLzC9K0oyXdJo7nve/bynv/KwUsxbiTlzAM=",
    version = "v0.0.0-20200831202436-30c9fc6549a9",
)

go_repository(
    name = "com_github_ajstarks_deck_generate",
    importpath = "github.com/ajstarks/deck/generate",
    sum = "h1:iXUgAaqDcIUGbRoy2TdeofRG/j1zpGRSEmNK05T+bi8=",
    version = "v0.0.0-20210309230005-c3f852c02e19",
)

go_repository(
    name = "com_github_ajstarks_svgo",
    importpath = "github.com/ajstarks/svgo",
    sum = "h1:slYM766cy2nI3BwyRiyQj/Ud48djTMtMebDqepE95rw=",
    version = "v0.0.0-20211024235047-1546f124cd8b",
)

go_repository(
    name = "com_github_boombuler_barcode",
    importpath = "github.com/boombuler/barcode",
    sum = "h1:NDBbPmhS+EqABEs5Kg3n/5ZNjy73Pz7SIV+KCeqyXcs=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_fogleman_gg",
    importpath = "github.com/fogleman/gg",
    sum = "h1:/7zJX8F6AaYQc57WQCyN9cAIz+4bCJGO9B+dyW29am8=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_go_fonts_dejavu",
    importpath = "github.com/go-fonts/dejavu",
    sum = "h1:JSajPXURYqpr+Cu8U9bt8K+XcACIHWqWrvWCKyeFmVQ=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_go_fonts_latin_modern",
    importpath = "github.com/go-fonts/latin-modern",
    sum = "h1:5/Tv1Ek/QCr20C6ZOz15vw3g7GELYL98KWr8Hgo+3vk=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_go_fonts_liberation",
    importpath = "github.com/go-fonts/liberation",
    sum = "h1:jAkAWJP4S+OsrPLZM4/eC9iW7CtHy+HBXrEwZXWo5VM=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_go_fonts_stix",
    importpath = "github.com/go-fonts/stix",
    sum = "h1:UlZlgrvvmT/58o573ot7NFw0vZasZ5I6bcIft/oMdgg=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_go_latex_latex",
    importpath = "github.com/go-latex/latex",
    sum = "h1:6zl3BbBhdnMkpSj2YY30qV3gDcVBGtFgVsV3+/i+mKQ=",
    version = "v0.0.0-20210823091927-c0d11ff05a81",
)

go_repository(
    name = "com_github_go_pdf_fpdf",
    importpath = "github.com/go-pdf/fpdf",
    sum = "h1:MlgtGIfsdMEEQJr2le6b/HNr1ZlQwxyWr77r2aj2U/8=",
    version = "v0.6.0",
)

go_repository(
    name = "com_github_golang_freetype",
    importpath = "github.com/golang/freetype",
    sum = "h1:DACJavvAHhabrF08vX0COfcOBJRhZ8lUbR+ZWIs0Y5g=",
    version = "v0.0.0-20170609003504-e2365dfdc4a0",
)

go_repository(
    name = "com_github_jung_kurt_gofpdf",
    importpath = "github.com/jung-kurt/gofpdf",
    sum = "h1:PJr+ZMXIecYc1Ey2zucXdR73SMBtgjPgwa31099IMv0=",
    version = "v1.0.3-0.20190309125859-24315acbbda5",
)

go_repository(
    name = "com_github_metalblueberry_go_plotly",
    importpath = "github.com/MetalBlueberry/go-plotly",
    sum = "h1:ld/FLZIwLmPdv09ljANonwEqSoI1uNn7myLYAVjBQ48=",
    version = "v0.4.0",
)

go_repository(
    name = "com_github_phpdave11_gofpdf",
    importpath = "github.com/phpdave11/gofpdf",
    sum = "h1:KPKiIbfwbvC/wOncwhrpRdXVj2CZTCFlw4wnoyjtHfQ=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_phpdave11_gofpdi",
    importpath = "github.com/phpdave11/gofpdi",
    sum = "h1:o61duiW8M9sMlkVXWlvP92sZJtGKENvW3VExs6dZukQ=",
    version = "v1.0.13",
)

go_repository(
    name = "com_github_pkg_browser",
    importpath = "github.com/pkg/browser",
    sum = "h1:49lOXmGaUpV9Fz3gd7TFZY106KVlPVa5jcYD1gaQf98=",
    version = "v0.0.0-20180916011732-0a3d74bf9ce4",
)

go_repository(
    name = "com_github_ruudk_golang_pdf417",
    importpath = "github.com/ruudk/golang-pdf417",
    sum = "h1:K1Xf3bKttbF+koVGaX5xngRIZ5bVjbmPnaxE/dR08uY=",
    version = "v0.0.0-20201230142125-a7e3863a1245",
)

go_repository(
    name = "ht_sr_git_sbinet_gg",
    importpath = "git.sr.ht/~sbinet/gg",
    sum = "h1:LNhjNn8DerC8f9DHLz6lS0YYul/b602DUxDgGkd/Aik=",
    version = "v0.3.1",
)

go_repository(
    name = "io_rsc_pdf",
    importpath = "rsc.io/pdf",
    sum = "h1:k1MczvYDUvJBe93bYd7wrZLLUEcLZAuF824/I4e5Xr4=",
    version = "v0.1.1",
)

go_repository(
    name = "org_gioui",
    importpath = "gioui.org",
    sum = "h1:K72hopUosKG3ntOPNG4OzzbuhxGuVf06fa2la1/H/Ho=",
    version = "v0.0.0-20210308172011-57750fc8a0a6",
)

go_repository(
    name = "org_golang_x_exp_shiny",
    importpath = "golang.org/x/exp/shiny",
    sum = "h1:pkl1Ko5DrhA4ezwKwdnmO7H1sKmMy9qLuYKRjS7SlmE=",
    version = "v0.0.0-20220722155223-a9213eeb770e",
)

go_repository(
    name = "org_gonum_v1_plot",
    importpath = "gonum.org/v1/plot",
    sum = "h1:y1ZNmfz/xHuHvtgFe8USZVyykQo5ERXPnspQNVK15Og=",
    version = "v0.12.0",
)

go_repository(
    name = "com_github_onsi_ginkgo_v2",
    importpath = "github.com/onsi/ginkgo/v2",
    sum = "h1:BA2GMJOtfGAfagzYtrAlufIP0lq6QERkFmHLMLPwFSU=",
    version = "v2.9.2",
)

go_repository(
    name = "com_github_antlr_antlr4_runtime_go_antlr",
    importpath = "github.com/antlr/antlr4/runtime/Go/antlr",
    sum = "h1:yL7+Jz0jTC6yykIK/Wh74gnTJnrGr5AyrNMXuA0gves=",
    version = "v1.4.10",
)

go_repository(
    name = "com_github_benbjohnson_clock",
    importpath = "github.com/benbjohnson/clock",
    sum = "h1:Q92kusRqC1XV2MjkWETPvjJVqKetz1OzxZB7mHJLju8=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_blang_semver_v4",
    importpath = "github.com/blang/semver/v4",
    sum = "h1:1PFHFE6yCCTv8C1TeyNNarDzntLi7wMI5i/pzqYIsAM=",
    version = "v4.0.0",
)

go_repository(
    name = "com_github_cenkalti_backoff_v4",
    importpath = "github.com/cenkalti/backoff/v4",
    sum = "h1:cFAlzYUlVYDysBEH2T5hyJZMh3+5+WCBvSnK6Q8UtC4=",
    version = "v4.1.3",
)

go_repository(
    name = "com_github_certifi_gocertifi",
    importpath = "github.com/certifi/gocertifi",
    sum = "h1:uH66TXeswKn5PW5zdZ39xEwfS9an067BirqA+P4QaLI=",
    version = "v0.0.0-20200922220541-2c3bb06c6054",
)

go_repository(
    name = "com_github_cncf_xds_go",
    importpath = "github.com/cncf/xds/go",
    sum = "h1:58f1tJ1ra+zFINPlwLWvQsR9CzAKt2e+EWV2yX9oXQ4=",
    version = "v0.0.0-20230310173818-32f1caf87195",
)

go_repository(
    name = "com_github_cockroachdb_errors",
    importpath = "github.com/cockroachdb/errors",
    sum = "h1:Lap807SXTH5tri2TivECb/4abUkMZC9zRoLarvcKDqs=",
    version = "v1.2.4",
)

go_repository(
    name = "com_github_cockroachdb_logtags",
    importpath = "github.com/cockroachdb/logtags",
    sum = "h1:o/kfcElHqOiXqcou5a3rIlMc7oJbMQkeLk0VQJ7zgqY=",
    version = "v0.0.0-20190617123548-eb05cc24525f",
)

go_repository(
    name = "com_github_coreos_go_systemd_v22",
    importpath = "github.com/coreos/go-systemd/v22",
    sum = "h1:D9/bQk5vlXQFZ6Kwuu6zaiXJ9oTPe68++AzAJc1DzSI=",
    version = "v22.3.2",
)

go_repository(
    name = "com_github_emicklei_go_restful_v3",
    importpath = "github.com/emicklei/go-restful/v3",
    sum = "h1:XwGDlfxEnQZzuopoqxwSEllNcCOM9DhhFyhFIIGKwxE=",
    version = "v3.9.0",
)

go_repository(
    name = "com_github_evanphx_json_patch_v5",
    importpath = "github.com/evanphx/json-patch/v5",
    sum = "h1:b91NhWfaz02IuVxO9faSllyAtNXHMPkC5J8sJCLunww=",
    version = "v5.6.0",
)

go_repository(
    name = "com_github_felixge_httpsnoop",
    importpath = "github.com/felixge/httpsnoop",
    sum = "h1:s/nj+GCswXYzN5v2DpNMuMQYe+0DDwt5WVCU6CWBdXk=",
    version = "v1.0.3",
)

go_repository(
    name = "com_github_getsentry_raven_go",
    importpath = "github.com/getsentry/raven-go",
    sum = "h1:no+xWJRb5ZI7eE8TWgIq1jLulQiIoLG0IfYxv5JYMGs=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_go_logr_stdr",
    importpath = "github.com/go-logr/stdr",
    sum = "h1:hSWxHoqTgW2S2qGc0LTAI563KZ5YKYRhT3MFKZMbjag=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_godbus_dbus_v5",
    importpath = "github.com/godbus/dbus/v5",
    sum = "h1:mkgN1ofwASrYnJ5W6U/BxG15eXXXjirgZc7CLqkcaro=",
    version = "v5.0.6",
)

go_repository(
    name = "com_github_google_cel_go",
    importpath = "github.com/google/cel-go",
    sum = "h1:kjeKudqV0OygrAqA9fX6J55S8gj+Jre2tckIm5RoG4M=",
    version = "v0.12.6",
)

go_repository(
    name = "com_github_google_gnostic",
    build_file_proto_mode = "disable",
    importpath = "github.com/google/gnostic",
    sum = "h1:ZK/5VhkoX835RikCHpSUJV9a+S3e1zLh59YnyWeBW+0=",
    version = "v0.6.9",
)

go_repository(
    name = "com_github_googleapis_enterprise_certificate_proxy",
    importpath = "github.com/googleapis/enterprise-certificate-proxy",
    sum = "h1:yk9/cqRKtT9wXZSsRH9aurXEpJX+U6FLtpYTdC3R06k=",
    version = "v0.2.3",
)

go_repository(
    name = "com_github_grpc_ecosystem_grpc_gateway_v2",
    importpath = "github.com/grpc-ecosystem/grpc-gateway/v2",
    sum = "h1:lLT7ZLSzGLI08vc9cpd+tYmNWjdKDqyr/2L+f6U12Fk=",
    version = "v2.11.3",
)

go_repository(
    name = "com_github_josharian_intern",
    importpath = "github.com/josharian/intern",
    sum = "h1:vlS4z54oSdjm0bgjRigI+G1HpF+tI+9rE5LLzOg8HmY=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    importpath = "github.com/opentracing/opentracing-go",
    sum = "h1:pWlfV3Bxv7k65HYwkikxat0+s3pV4bsqf19k25Ur8rU=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_stoewer_go_strcase",
    importpath = "github.com/stoewer/go-strcase",
    sum = "h1:Z2iHWqGXH00XYgqDmNgQbIBxf3wrNq0F3feEy0ainaU=",
    version = "v1.2.0",
)

go_repository(
    name = "com_google_cloud_go_accessapproval",
    importpath = "cloud.google.com/go/accessapproval",
    sum = "h1:x0cEHro/JFPd7eS4BlEWNTMecIj2HdXjOVB5BtvwER0=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_accesscontextmanager",
    importpath = "cloud.google.com/go/accesscontextmanager",
    sum = "h1:MG60JgnEoawHJrbWw0jGdv6HLNSf6gQvYRiXpuzqgEA=",
    version = "v1.7.0",
)

go_repository(
    name = "com_google_cloud_go_aiplatform",
    importpath = "cloud.google.com/go/aiplatform",
    sum = "h1:zTw+suCVchgZyO+k847wjzdVjWmrAuehxdvcZvJwfGg=",
    version = "v1.37.0",
)

go_repository(
    name = "com_google_cloud_go_analytics",
    importpath = "cloud.google.com/go/analytics",
    sum = "h1:LqAo3tAh2FU9+w/r7vc3hBjU23Kv7GhO/PDIW7kIYgM=",
    version = "v0.19.0",
)

go_repository(
    name = "com_google_cloud_go_apigateway",
    importpath = "cloud.google.com/go/apigateway",
    sum = "h1:ZI9mVO7x3E9RK/BURm2p1aw9YTBSCQe3klmyP1WxWEg=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_apigeeconnect",
    importpath = "cloud.google.com/go/apigeeconnect",
    sum = "h1:sWOmgDyAsi1AZ48XRHcATC0tsi9SkPT7DA/+VCfkaeA=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_apigeeregistry",
    importpath = "cloud.google.com/go/apigeeregistry",
    sum = "h1:E43RdhhCxdlV+I161gUY2rI4eOaMzHTA5kNkvRsFXvc=",
    version = "v0.6.0",
)

go_repository(
    name = "com_google_cloud_go_apikeys",
    importpath = "cloud.google.com/go/apikeys",
    sum = "h1:B9CdHFZTFjVti89tmyXXrO+7vSNo2jvZuHG8zD5trdQ=",
    version = "v0.6.0",
)

go_repository(
    name = "com_google_cloud_go_appengine",
    importpath = "cloud.google.com/go/appengine",
    sum = "h1:aBGDKmRIaRRoWJ2tAoN0oVSHoWLhtO9aj/NvUyP4aYs=",
    version = "v1.7.1",
)

go_repository(
    name = "com_google_cloud_go_area120",
    importpath = "cloud.google.com/go/area120",
    sum = "h1:ugckkFh4XkHJMPhTIx0CyvdoBxmOpMe8rNs4Ok8GAag=",
    version = "v0.7.1",
)

go_repository(
    name = "com_google_cloud_go_artifactregistry",
    importpath = "cloud.google.com/go/artifactregistry",
    sum = "h1:o1Q80vqEB6Qp8WLEH3b8FBLNUCrGQ4k5RFj0sn/sgO8=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_asset",
    importpath = "cloud.google.com/go/asset",
    sum = "h1:YAsssO08BqZ6mncbb6FPlj9h6ACS7bJQUOlzciSfbNk=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_assuredworkloads",
    importpath = "cloud.google.com/go/assuredworkloads",
    sum = "h1:VLGnVFta+N4WM+ASHbhc14ZOItOabDLH1MSoDv+Xuag=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_automl",
    importpath = "cloud.google.com/go/automl",
    sum = "h1:50VugllC+U4IGl3tDNcZaWvApHBTrn/TvyHDJ0wM+Uw=",
    version = "v1.12.0",
)

go_repository(
    name = "com_google_cloud_go_baremetalsolution",
    importpath = "cloud.google.com/go/baremetalsolution",
    sum = "h1:2AipdYXL0VxMboelTTw8c1UJ7gYu35LZYUbuRv9Q28s=",
    version = "v0.5.0",
)

go_repository(
    name = "com_google_cloud_go_batch",
    importpath = "cloud.google.com/go/batch",
    sum = "h1:YbMt0E6BtqeD5FvSv1d56jbVsWEzlGm55lYte+M6Mzs=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_beyondcorp",
    importpath = "cloud.google.com/go/beyondcorp",
    sum = "h1:UkY2BTZkEUAVrgqnSdOJ4p3y9ZRBPEe1LkjgC8Bj/Pc=",
    version = "v0.5.0",
)

go_repository(
    name = "com_google_cloud_go_billing",
    importpath = "cloud.google.com/go/billing",
    sum = "h1:JYj28UYF5w6VBAh0gQYlgHJ/OD1oA+JgW29YZQU+UHM=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_binaryauthorization",
    importpath = "cloud.google.com/go/binaryauthorization",
    sum = "h1:d3pMDBCCNivxt5a4eaV7FwL7cSH0H7RrEnFrTb1QKWs=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_certificatemanager",
    importpath = "cloud.google.com/go/certificatemanager",
    sum = "h1:5C5UWeSt8Jkgp7OWn2rCkLmYurar/vIWIoSQ2+LaTOc=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_channel",
    importpath = "cloud.google.com/go/channel",
    sum = "h1:GpcQY5UJKeOekYgsX3QXbzzAc/kRGtBq43fTmyKe6Uw=",
    version = "v1.12.0",
)

go_repository(
    name = "com_google_cloud_go_cloudbuild",
    importpath = "cloud.google.com/go/cloudbuild",
    sum = "h1:GHQCjV4WlPPVU/j3Rlpc8vNIDwThhd1U9qSY/NPZdko=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_clouddms",
    importpath = "cloud.google.com/go/clouddms",
    sum = "h1:E7v4TpDGUyEm1C/4KIrpVSOCTm0P6vWdHT0I4mostRA=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_cloudtasks",
    importpath = "cloud.google.com/go/cloudtasks",
    sum = "h1:uK5k6abf4yligFgYFnG0ni8msai/dSv6mDmiBulU0hU=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_compute",
    importpath = "cloud.google.com/go/compute",
    sum = "h1:am86mquDUgjGNWxiGn+5PGLbmgiWXlE/yNWpIpNvuXY=",
    version = "v1.19.1",
)

go_repository(
    name = "com_google_cloud_go_compute_metadata",
    importpath = "cloud.google.com/go/compute/metadata",
    sum = "h1:mg4jlk7mCAj6xXp9UJ4fjI9VUI5rubuGBW5aJ7UnBMY=",
    version = "v0.2.3",
)

go_repository(
    name = "com_google_cloud_go_contactcenterinsights",
    importpath = "cloud.google.com/go/contactcenterinsights",
    sum = "h1:jXIpfcH/VYSE1SYcPzO0n1VVb+sAamiLOgCw45JbOQk=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_container",
    importpath = "cloud.google.com/go/container",
    sum = "h1:NKlY/wCDapfVZlbVVaeuu2UZZED5Dy1z4Zx1KhEzm8c=",
    version = "v1.15.0",
)

go_repository(
    name = "com_google_cloud_go_containeranalysis",
    importpath = "cloud.google.com/go/containeranalysis",
    sum = "h1:EQ4FFxNaEAg8PqQCO7bVQfWz9NVwZCUKaM1b3ycfx3U=",
    version = "v0.9.0",
)

go_repository(
    name = "com_google_cloud_go_datacatalog",
    importpath = "cloud.google.com/go/datacatalog",
    sum = "h1:4H5IJiyUE0X6ShQBqgFFZvGGcrwGVndTwUSLP4c52gw=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_dataflow",
    importpath = "cloud.google.com/go/dataflow",
    sum = "h1:eYyD9o/8Nm6EttsKZaEGD84xC17bNgSKCu0ZxwqUbpg=",
    version = "v0.8.0",
)

go_repository(
    name = "com_google_cloud_go_dataform",
    importpath = "cloud.google.com/go/dataform",
    sum = "h1:Dyk+fufup1FR6cbHjFpMuP4SfPiF3LI3JtoIIALoq48=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_datafusion",
    importpath = "cloud.google.com/go/datafusion",
    sum = "h1:sZjRnS3TWkGsu1LjYPFD/fHeMLZNXDK6PDHi2s2s/bk=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_datalabeling",
    importpath = "cloud.google.com/go/datalabeling",
    sum = "h1:ch4qA2yvddGRUrlfwrNJCr79qLqhS9QBwofPHfFlDIk=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_dataplex",
    importpath = "cloud.google.com/go/dataplex",
    sum = "h1:RvoZ5T7gySwm1CHzAw7yY1QwwqaGswunmqEssPxU/AM=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_dataproc",
    importpath = "cloud.google.com/go/dataproc",
    sum = "h1:W47qHL3W4BPkAIbk4SWmIERwsWBaNnWm0P2sdx3YgGU=",
    version = "v1.12.0",
)

go_repository(
    name = "com_google_cloud_go_dataqna",
    importpath = "cloud.google.com/go/dataqna",
    sum = "h1:yFzi/YU4YAdjyo7pXkBE2FeHbgz5OQQBVDdbErEHmVQ=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_datastream",
    importpath = "cloud.google.com/go/datastream",
    sum = "h1:BBCBTnWMDwwEzQQmipUXxATa7Cm7CA/gKjKcR2w35T0=",
    version = "v1.7.0",
)

go_repository(
    name = "com_google_cloud_go_deploy",
    importpath = "cloud.google.com/go/deploy",
    sum = "h1:otshdKEbmsi1ELYeCKNYppwV0UH5xD05drSdBm7ouTk=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_dialogflow",
    importpath = "cloud.google.com/go/dialogflow",
    sum = "h1:uVlKKzp6G/VtSW0E7IH1Y5o0H48/UOCmqksG2riYCwQ=",
    version = "v1.32.0",
)

go_repository(
    name = "com_google_cloud_go_dlp",
    importpath = "cloud.google.com/go/dlp",
    sum = "h1:1JoJqezlgu6NWCroBxr4rOZnwNFILXr4cB9dMaSKO4A=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_documentai",
    importpath = "cloud.google.com/go/documentai",
    sum = "h1:KM3Xh0QQyyEdC8Gs2vhZfU+rt6OCPF0dwVwxKgLmWfI=",
    version = "v1.18.0",
)

go_repository(
    name = "com_google_cloud_go_domains",
    importpath = "cloud.google.com/go/domains",
    sum = "h1:2ti/o9tlWL4N+wIuWUNH+LbfgpwxPr8J1sv9RHA4bYQ=",
    version = "v0.8.0",
)

go_repository(
    name = "com_google_cloud_go_edgecontainer",
    importpath = "cloud.google.com/go/edgecontainer",
    sum = "h1:O0YVE5v+O0Q/ODXYsQHmHb+sYM8KNjGZw2pjX2Ws41c=",
    version = "v1.0.0",
)

go_repository(
    name = "com_google_cloud_go_errorreporting",
    importpath = "cloud.google.com/go/errorreporting",
    sum = "h1:kj1XEWMu8P0qlLhm3FwcaFsUvXChV/OraZwA70trRR0=",
    version = "v0.3.0",
)

go_repository(
    name = "com_google_cloud_go_essentialcontacts",
    importpath = "cloud.google.com/go/essentialcontacts",
    sum = "h1:gIzEhCoOT7bi+6QZqZIzX1Erj4SswMPIteNvYVlu+pM=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_eventarc",
    importpath = "cloud.google.com/go/eventarc",
    sum = "h1:fsJmNeqvqtk74FsaVDU6cH79lyZNCYP8Rrv7EhaB/PU=",
    version = "v1.11.0",
)

go_repository(
    name = "com_google_cloud_go_filestore",
    importpath = "cloud.google.com/go/filestore",
    sum = "h1:ckTEXN5towyTMu4q0uQ1Mde/JwTHur0gXs8oaIZnKfw=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_functions",
    importpath = "cloud.google.com/go/functions",
    sum = "h1:pPDqtsXG2g9HeOQLoquLbmvmb82Y4Ezdo1GXuotFoWg=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_gaming",
    importpath = "cloud.google.com/go/gaming",
    sum = "h1:7vEhFnZmd931Mo7sZ6pJy7uQPDxF7m7v8xtBheG08tc=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_gkebackup",
    importpath = "cloud.google.com/go/gkebackup",
    sum = "h1:za3QZvw6ujR0uyqkhomKKKNoXDyqYGPJies3voUK8DA=",
    version = "v0.4.0",
)

go_repository(
    name = "com_google_cloud_go_gkeconnect",
    importpath = "cloud.google.com/go/gkeconnect",
    sum = "h1:gXYKciHS/Lgq0GJ5Kc9SzPA35NGc3yqu6SkjonpEr2Q=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_gkehub",
    importpath = "cloud.google.com/go/gkehub",
    sum = "h1:TqCSPsEBQ6oZSJgEYZ3XT8x2gUadbvfwI32YB0kuHCs=",
    version = "v0.12.0",
)

go_repository(
    name = "com_google_cloud_go_gkemulticloud",
    importpath = "cloud.google.com/go/gkemulticloud",
    sum = "h1:8I84Q4vl02rJRsFiinBxl7WCozfdLlUVBQuSrqr9Wtk=",
    version = "v0.5.0",
)

go_repository(
    name = "com_google_cloud_go_gsuiteaddons",
    importpath = "cloud.google.com/go/gsuiteaddons",
    sum = "h1:1mvhXqJzV0Vg5Fa95QwckljODJJfDFXV4pn+iL50zzA=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_iam",
    importpath = "cloud.google.com/go/iam",
    sum = "h1:+CmB+K0J/33d0zSQ9SlFWUeCCEn5XJA0ZMZ3pHE9u8k=",
    version = "v0.13.0",
)

go_repository(
    name = "com_google_cloud_go_iap",
    importpath = "cloud.google.com/go/iap",
    sum = "h1:PxVHFuMxmSZyfntKXHXhd8bo82WJ+LcATenq7HLdVnU=",
    version = "v1.7.1",
)

go_repository(
    name = "com_google_cloud_go_ids",
    importpath = "cloud.google.com/go/ids",
    sum = "h1:fodnCDtOXuMmS8LTC2y3h8t24U8F3eKWfhi+3LY6Qf0=",
    version = "v1.3.0",
)

go_repository(
    name = "com_google_cloud_go_iot",
    importpath = "cloud.google.com/go/iot",
    sum = "h1:39W5BFSarRNZfVG0eXI5LYux+OVQT8GkgpHCnrZL2vM=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_kms",
    importpath = "cloud.google.com/go/kms",
    sum = "h1:7hm1bRqGCA1GBRQUrp831TwJ9TWhP+tvLuP497CQS2g=",
    version = "v1.10.1",
)

go_repository(
    name = "com_google_cloud_go_language",
    importpath = "cloud.google.com/go/language",
    sum = "h1:7Ulo2mDk9huBoBi8zCE3ONOoBrL6UXfAI71CLQ9GEIM=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_lifesciences",
    importpath = "cloud.google.com/go/lifesciences",
    sum = "h1:uWrMjWTsGjLZpCTWEAzYvyXj+7fhiZST45u9AgasasI=",
    version = "v0.8.0",
)

go_repository(
    name = "com_google_cloud_go_longrunning",
    importpath = "cloud.google.com/go/longrunning",
    sum = "h1:v+yFJOfKC3yZdY6ZUI933pIYdhyhV8S3NpWrXWmg7jM=",
    version = "v0.4.1",
)

go_repository(
    name = "com_google_cloud_go_managedidentities",
    importpath = "cloud.google.com/go/managedidentities",
    sum = "h1:ZRQ4k21/jAhrHBVKl/AY7SjgzeJwG1iZa+mJ82P+VNg=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_maps",
    importpath = "cloud.google.com/go/maps",
    sum = "h1:mv9YaczD4oZBZkM5XJl6fXQ984IkJNHPwkc8MUsdkBo=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_mediatranslation",
    importpath = "cloud.google.com/go/mediatranslation",
    sum = "h1:anPxH+/WWt8Yc3EdoEJhPMBRF7EhIdz426A+tuoA0OU=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_memcache",
    importpath = "cloud.google.com/go/memcache",
    sum = "h1:8/VEmWCpnETCrBwS3z4MhT+tIdKgR1Z4Tr2tvYH32rg=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_metastore",
    importpath = "cloud.google.com/go/metastore",
    sum = "h1:QCFhZVe2289KDBQ7WxaHV2rAmPrmRAdLC6gbjUd3HPo=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_monitoring",
    importpath = "cloud.google.com/go/monitoring",
    sum = "h1:2qsrgXGVoRXpP7otZ14eE1I568zAa92sJSDPyOJvwjM=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_networkconnectivity",
    importpath = "cloud.google.com/go/networkconnectivity",
    sum = "h1:ZD6b4Pk1jEtp/cx9nx0ZYcL3BKqDa+KixNDZ6Bjs1B8=",
    version = "v1.11.0",
)

go_repository(
    name = "com_google_cloud_go_networkmanagement",
    importpath = "cloud.google.com/go/networkmanagement",
    sum = "h1:8KWEUNGcpSX9WwZXq7FtciuNGPdPdPN/ruDm769yAEM=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_networksecurity",
    importpath = "cloud.google.com/go/networksecurity",
    sum = "h1:sOc42Ig1K2LiKlzG71GUVloeSJ0J3mffEBYmvu+P0eo=",
    version = "v0.8.0",
)

go_repository(
    name = "com_google_cloud_go_notebooks",
    importpath = "cloud.google.com/go/notebooks",
    sum = "h1:Kg2K3K7CbSXYJHZ1aGQpf1xi5x2GUvQWf2sFVuiZh8M=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_optimization",
    importpath = "cloud.google.com/go/optimization",
    sum = "h1:dj8O4VOJRB4CUwZXdmwNViH1OtI0WtWL867/lnYH248=",
    version = "v1.3.1",
)

go_repository(
    name = "com_google_cloud_go_orchestration",
    importpath = "cloud.google.com/go/orchestration",
    sum = "h1:Vw+CEXo8M/FZ1rb4EjcLv0gJqqw89b7+g+C/EmniTb8=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_orgpolicy",
    importpath = "cloud.google.com/go/orgpolicy",
    sum = "h1:XDriMWug7sd0kYT1QKofRpRHzjad0bK8Q8uA9q+XrU4=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_osconfig",
    importpath = "cloud.google.com/go/osconfig",
    sum = "h1:PkSQx4OHit5xz2bNyr11KGcaFccL5oqglFPdTboyqwQ=",
    version = "v1.11.0",
)

go_repository(
    name = "com_google_cloud_go_oslogin",
    importpath = "cloud.google.com/go/oslogin",
    sum = "h1:whP7vhpmc+ufZa90eVpkfbgzJRK/Xomjz+XCD4aGwWw=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_phishingprotection",
    importpath = "cloud.google.com/go/phishingprotection",
    sum = "h1:l6tDkT7qAEV49MNEJkEJTB6vOO/onbSOcNtAT09HPuA=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_policytroubleshooter",
    importpath = "cloud.google.com/go/policytroubleshooter",
    sum = "h1:yKAGC4p9O61ttZUswaq9GAn1SZnEzTd0vUYXD7ZBT7Y=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_privatecatalog",
    importpath = "cloud.google.com/go/privatecatalog",
    sum = "h1:EPEJ1DpEGXLDnmc7mnCAqFmkwUJbIsaLAiLHVOkkwtc=",
    version = "v0.8.0",
)

go_repository(
    name = "com_google_cloud_go_pubsublite",
    importpath = "cloud.google.com/go/pubsublite",
    sum = "h1:cb9fsrtpINtETHiJ3ECeaVzrfIVhcGjhhJEjybHXHao=",
    version = "v1.7.0",
)

go_repository(
    name = "com_google_cloud_go_recaptchaenterprise_v2",
    importpath = "cloud.google.com/go/recaptchaenterprise/v2",
    sum = "h1:6iOCujSNJ0YS7oNymI64hXsjGq60T4FK1zdLugxbzvU=",
    version = "v2.7.0",
)

go_repository(
    name = "com_google_cloud_go_recommendationengine",
    importpath = "cloud.google.com/go/recommendationengine",
    sum = "h1:VibRFCwWXrFebEWKHfZAt2kta6pS7Tlimsnms0fjv7k=",
    version = "v0.7.0",
)

go_repository(
    name = "com_google_cloud_go_recommender",
    importpath = "cloud.google.com/go/recommender",
    sum = "h1:ZnFRY5R6zOVk2IDS1Jbv5Bw+DExCI5rFumsTnMXiu/A=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_redis",
    importpath = "cloud.google.com/go/redis",
    sum = "h1:JoAd3SkeDt3rLFAAxEvw6wV4t+8y4ZzfZcZmddqphQ8=",
    version = "v1.11.0",
)

go_repository(
    name = "com_google_cloud_go_resourcemanager",
    importpath = "cloud.google.com/go/resourcemanager",
    sum = "h1:NRM0p+RJkaQF9Ee9JMnUV9BQ2QBIOq/v8M+Pbv/wmCs=",
    version = "v1.7.0",
)

go_repository(
    name = "com_google_cloud_go_resourcesettings",
    importpath = "cloud.google.com/go/resourcesettings",
    sum = "h1:8Dua37kQt27CCWHm4h/Q1XqCF6ByD7Ouu49xg95qJzI=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_retail",
    importpath = "cloud.google.com/go/retail",
    sum = "h1:1Dda2OpFNzIb4qWgFZjYlpP7sxX3aLeypKG6A3H4Yys=",
    version = "v1.12.0",
)

go_repository(
    name = "com_google_cloud_go_run",
    importpath = "cloud.google.com/go/run",
    sum = "h1:ydJQo+k+MShYnBfhaRHSZYeD/SQKZzZLAROyfpeD9zw=",
    version = "v0.9.0",
)

go_repository(
    name = "com_google_cloud_go_scheduler",
    importpath = "cloud.google.com/go/scheduler",
    sum = "h1:NpQAHtx3sulByTLe2dMwWmah8PWgeoieFPpJpArwFV0=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_secretmanager",
    importpath = "cloud.google.com/go/secretmanager",
    sum = "h1:pu03bha7ukxF8otyPKTFdDz+rr9sE3YauS5PliDXK60=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_security",
    importpath = "cloud.google.com/go/security",
    sum = "h1:PYvDxopRQBfYAXKAuDpFCKBvDOWPWzp9k/H5nB3ud3o=",
    version = "v1.13.0",
)

go_repository(
    name = "com_google_cloud_go_securitycenter",
    importpath = "cloud.google.com/go/securitycenter",
    sum = "h1:AF3c2s3awNTMoBtMX3oCUoOMmGlYxGOeuXSYHNBkf14=",
    version = "v1.19.0",
)

go_repository(
    name = "com_google_cloud_go_servicecontrol",
    importpath = "cloud.google.com/go/servicecontrol",
    sum = "h1:d0uV7Qegtfaa7Z2ClDzr9HJmnbJW7jn0WhZ7wOX6hLE=",
    version = "v1.11.1",
)

go_repository(
    name = "com_google_cloud_go_servicedirectory",
    importpath = "cloud.google.com/go/servicedirectory",
    sum = "h1:SJwk0XX2e26o25ObYUORXx6torSFiYgsGkWSkZgkoSU=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_servicemanagement",
    importpath = "cloud.google.com/go/servicemanagement",
    sum = "h1:fopAQI/IAzlxnVeiKn/8WiV6zKndjFkvi+gzu+NjywY=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_serviceusage",
    importpath = "cloud.google.com/go/serviceusage",
    sum = "h1:rXyq+0+RSIm3HFypctp7WoXxIA563rn206CfMWdqXX4=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_shell",
    importpath = "cloud.google.com/go/shell",
    sum = "h1:wT0Uw7ib7+AgZST9eCDygwTJn4+bHMDtZo5fh7kGWDU=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_spanner",
    importpath = "cloud.google.com/go/spanner",
    sum = "h1:7VdjZ8zj4sHbDw55atp5dfY6kn1j9sam9DRNpPQhqR4=",
    version = "v1.45.0",
)

go_repository(
    name = "com_google_cloud_go_speech",
    importpath = "cloud.google.com/go/speech",
    sum = "h1:JEVoWGNnTF128kNty7T4aG4eqv2z86yiMJPT9Zjp+iw=",
    version = "v1.15.0",
)

go_repository(
    name = "com_google_cloud_go_storagetransfer",
    importpath = "cloud.google.com/go/storagetransfer",
    sum = "h1:5T+PM+3ECU3EY2y9Brv0Sf3oka8pKmsCfpQ07+91G9o=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_talent",
    importpath = "cloud.google.com/go/talent",
    sum = "h1:nI9sVZPjMKiO2q3Uu0KhTDVov3Xrlpt63fghP9XjyEM=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_texttospeech",
    importpath = "cloud.google.com/go/texttospeech",
    sum = "h1:H4g1ULStsbVtalbZGktyzXzw6jP26RjVGYx9RaYjBzc=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_tpu",
    importpath = "cloud.google.com/go/tpu",
    sum = "h1:/34T6CbSi+kTv5E19Q9zbU/ix8IviInZpzwz3rsFE+A=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_trace",
    importpath = "cloud.google.com/go/trace",
    sum = "h1:olxC0QHC59zgJVALtgqfD9tGk0lfeCP5/AGXL3Px/no=",
    version = "v1.9.0",
)

go_repository(
    name = "com_google_cloud_go_translate",
    importpath = "cloud.google.com/go/translate",
    sum = "h1:GvLP4oQ4uPdChBmBaUSa/SaZxCdyWELtlAaKzpHsXdA=",
    version = "v1.7.0",
)

go_repository(
    name = "com_google_cloud_go_video",
    importpath = "cloud.google.com/go/video",
    sum = "h1:upIbnGI0ZgACm58HPjAeBMleW3sl5cT84AbYQ8PWOgM=",
    version = "v1.15.0",
)

go_repository(
    name = "com_google_cloud_go_videointelligence",
    importpath = "cloud.google.com/go/videointelligence",
    sum = "h1:Uh5BdoET8XXqXX2uXIahGb+wTKbLkGH7s4GXR58RrG8=",
    version = "v1.10.0",
)

go_repository(
    name = "com_google_cloud_go_vision_v2",
    importpath = "cloud.google.com/go/vision/v2",
    sum = "h1:8C8RXUJoflCI4yVdqhTy9tRyygSHmp60aP363z23HKg=",
    version = "v2.7.0",
)

go_repository(
    name = "com_google_cloud_go_vmmigration",
    importpath = "cloud.google.com/go/vmmigration",
    sum = "h1:Azs5WKtfOC8pxvkyrDvt7J0/4DYBch0cVbuFfCCFt5k=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_vmwareengine",
    importpath = "cloud.google.com/go/vmwareengine",
    sum = "h1:b0NBu7S294l0gmtrT0nOJneMYgZapr5x9tVWvgDoVEM=",
    version = "v0.3.0",
)

go_repository(
    name = "com_google_cloud_go_vpcaccess",
    importpath = "cloud.google.com/go/vpcaccess",
    sum = "h1:FOe6CuiQD3BhHJWt7E8QlbBcaIzVRddupwJlp7eqmn4=",
    version = "v1.6.0",
)

go_repository(
    name = "com_google_cloud_go_webrisk",
    importpath = "cloud.google.com/go/webrisk",
    sum = "h1:IY+L2+UwxcVm2zayMAtBhZleecdIFLiC+QJMzgb0kT0=",
    version = "v1.8.0",
)

go_repository(
    name = "com_google_cloud_go_websecurityscanner",
    importpath = "cloud.google.com/go/websecurityscanner",
    sum = "h1:AHC1xmaNMOZtNqxI9Rmm87IJEyPaRkOxeI0gpAacXGk=",
    version = "v1.5.0",
)

go_repository(
    name = "com_google_cloud_go_workflows",
    importpath = "cloud.google.com/go/workflows",
    sum = "h1:FfGp9w0cYnaKZJhUOMqCOJCYT/WlvYBfTQhFWV3sRKI=",
    version = "v1.10.0",
)

go_repository(
    name = "io_etcd_go_etcd_api_v3",
    importpath = "go.etcd.io/etcd/api/v3",
    sum = "h1:BX4JIbQ7hl7+jL+g+2j5UAr0o1bctCm6/Ct+ArBGkf0=",
    version = "v3.5.5",
)

go_repository(
    name = "io_etcd_go_etcd_client_pkg_v3",
    importpath = "go.etcd.io/etcd/client/pkg/v3",
    sum = "h1:9S0JUVvmrVl7wCF39iTQthdaaNIiAaQbmK75ogO6GU8=",
    version = "v3.5.5",
)

go_repository(
    name = "io_etcd_go_etcd_client_v2",
    importpath = "go.etcd.io/etcd/client/v2",
    sum = "h1:DktRP60//JJpnPC0VBymAN/7V71GHMdjDCBt4ZPXDjI=",
    version = "v2.305.5",
)

go_repository(
    name = "io_etcd_go_etcd_client_v3",
    importpath = "go.etcd.io/etcd/client/v3",
    sum = "h1:q++2WTJbUgpQu4B6hCuT7VkdwaTP7Qz6Daak3WzbrlI=",
    version = "v3.5.5",
)

go_repository(
    name = "io_etcd_go_etcd_pkg_v3",
    importpath = "go.etcd.io/etcd/pkg/v3",
    sum = "h1:Ablg7T7OkR+AeeeU32kdVhw/AGDsitkKPl7aW73ssjU=",
    version = "v3.5.5",
)

go_repository(
    name = "io_etcd_go_etcd_raft_v3",
    importpath = "go.etcd.io/etcd/raft/v3",
    sum = "h1:Ibz6XyZ60OYyRopu73lLM/P+qco3YtlZMOhnXNS051I=",
    version = "v3.5.5",
)

go_repository(
    name = "io_etcd_go_etcd_server_v3",
    importpath = "go.etcd.io/etcd/server/v3",
    sum = "h1:jNjYm/9s+f9A9r6+SC4RvNaz6AqixpOvhrFdT0PvIj0=",
    version = "v3.5.5",
)

go_repository(
    name = "io_k8s_kms",
    importpath = "k8s.io/kms",
    sum = "h1:JE0n4J4+8/Z+egvXz2BTJeJ9ecsm4ZSLKF7ttVXXm/4=",
    version = "v0.26.1",
)

go_repository(
    name = "io_k8s_sigs_json",
    importpath = "sigs.k8s.io/json",
    sum = "h1:EDPBXCAspyGV4jQlpZSudPeMmr1bNJefnuqLsRAsHZo=",
    version = "v0.0.0-20221116044647-bc3834ca7abd",
)

go_repository(
    name = "io_opentelemetry_go_contrib_instrumentation_google_golang_org_grpc_otelgrpc",
    importpath = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
    sum = "h1:xFSRQBbXF6VvYRf2lqMJXxoB72XI1K/azav8TekHHSw=",
    version = "v0.35.0",
)

go_repository(
    name = "io_opentelemetry_go_contrib_instrumentation_net_http_otelhttp",
    importpath = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
    sum = "h1:Ajldaqhxqw/gNzQA45IKFWLdG7jZuXX/wBW1d5qvbUI=",
    version = "v0.35.0",
)

go_repository(
    name = "io_opentelemetry_go_otel",
    importpath = "go.opentelemetry.io/otel",
    sum = "h1:1ZAKnNQKwBBxFtww/GwxNUyTf0AxkZzrukO8MeXqe4Y=",
    version = "v1.13.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_exporters_otlp_internal_retry",
    importpath = "go.opentelemetry.io/otel/exporters/otlp/internal/retry",
    sum = "h1:TaB+1rQhddO1sF71MpZOZAuSPW1klK2M8XxfrBMfK7Y=",
    version = "v1.10.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_exporters_otlp_otlptrace",
    importpath = "go.opentelemetry.io/otel/exporters/otlp/otlptrace",
    sum = "h1:pDDYmo0QadUPal5fwXoY1pmMpFcdyhXOmL5drCrI3vU=",
    version = "v1.10.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_exporters_otlp_otlptrace_otlptracegrpc",
    importpath = "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc",
    sum = "h1:KtiUEhQmj/Pa874bVYKGNVdq8NPKiacPbaRRtgXi+t4=",
    version = "v1.10.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_metric",
    importpath = "go.opentelemetry.io/otel/metric",
    sum = "h1:6SiklT+gfWAwWUR0meEMxQBtihpiEs4c+vL9spDTqUs=",
    version = "v0.31.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_sdk",
    importpath = "go.opentelemetry.io/otel/sdk",
    sum = "h1:BHib5g8MvdqS65yo2vV1s6Le42Hm6rrw08qU6yz5JaM=",
    version = "v1.13.0",
)

go_repository(
    name = "io_opentelemetry_go_otel_trace",
    importpath = "go.opentelemetry.io/otel/trace",
    sum = "h1:CBgRZ6ntv+Amuj1jDsMhZtlAPT6gbyIRdaIzFhfBSdY=",
    version = "v1.13.0",
)

go_repository(
    name = "io_opentelemetry_go_proto_otlp",
    importpath = "go.opentelemetry.io/proto/otlp",
    sum = "h1:IVN6GR+mhC4s5yfcTbmzHYODqvWAp3ZedA2SJPI1Nnw=",
    version = "v0.19.0",
)

go_repository(
    name = "org_golang_google_grpc_cmd_protoc_gen_go_grpc",
    importpath = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    sum = "h1:M1YKkFIboKNieVO5DLUEVzQfGwJD30Nv2jfUgzb5UcE=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_apache_arrow_go_v10",
    importpath = "github.com/apache/arrow/go/v10",
    sum = "h1:n9dERvixoC/1JjDmBcs9FPaEryoANa2sCgVFo6ez9cI=",
    version = "v10.0.1",
)

go_repository(
    name = "com_github_goccy_go_json",
    importpath = "github.com/goccy/go-json",
    sum = "h1:/pAaQDLHEoCq/5FFmSKBswWmK6H0e8g4159Kc/X/nqk=",
    version = "v0.9.11",
)

go_repository(
    name = "com_github_google_flatbuffers",
    importpath = "github.com/google/flatbuffers",
    sum = "h1:ivUb1cGomAB101ZM1T0nOiWz9pSrTMoa9+EiY7igmkM=",
    version = "v2.0.8+incompatible",
)

go_repository(
    name = "com_github_googleapis_go_type_adapters",
    importpath = "github.com/googleapis/go-type-adapters",
    sum = "h1:9XdMn+d/G57qq1s8dNc5IesGCXHf6V2HZ2JwRxfA2tA=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_googleapis_google_cloud_go_testing",
    importpath = "github.com/googleapis/google-cloud-go-testing",
    sum = "h1:tlyzajkF3030q6M8SvmJSemC9DTHL/xaMa18b65+JM4=",
    version = "v0.0.0-20200911160855-bcd43fbb19e8",
)

go_repository(
    name = "com_github_iancoleman_strcase",
    importpath = "github.com/iancoleman/strcase",
    sum = "h1:05I4QRnGpI0m37iZQRuskXh+w77mr6Z41lwQzuHLwW0=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_johncgriffin_overflow",
    importpath = "github.com/JohnCGriffin/overflow",
    sum = "h1:RGWPOewvKIROun94nF7v2cua9qP+thov/7M50KEoeSU=",
    version = "v0.0.0-20211019200055-46fa312c352c",
)

go_repository(
    name = "com_github_klauspost_asmfmt",
    importpath = "github.com/klauspost/asmfmt",
    sum = "h1:4Ri7ox3EwapiOjCki+hw14RyKk201CN4rzyCJRFLpK4=",
    version = "v1.3.2",
)

go_repository(
    name = "com_github_klauspost_cpuid_v2",
    importpath = "github.com/klauspost/cpuid/v2",
    sum = "h1:lgaqFMSdTdQYdZ04uHyN2d/eKdOMyi2YLSvlQIBFYa4=",
    version = "v2.0.9",
)

go_repository(
    name = "com_github_kr_fs",
    importpath = "github.com/kr/fs",
    sum = "h1:Jskdu9ieNAYnjxsi0LbQp1ulIKZV1LAFgK1tWhpZgl8=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_lyft_protoc_gen_star",
    importpath = "github.com/lyft/protoc-gen-star",
    sum = "h1:erE0rdztuaDq3bpGifD95wfoPrSZc95nGA6tbiNYh6M=",
    version = "v0.6.1",
)

go_repository(
    name = "com_github_minio_asm2plan9s",
    importpath = "github.com/minio/asm2plan9s",
    sum = "h1:AMFGa4R4MiIpspGNG7Z948v4n35fFGB3RR3G/ry4FWs=",
    version = "v0.0.0-20200509001527-cdd76441f9d8",
)

go_repository(
    name = "com_github_minio_c2goasm",
    importpath = "github.com/minio/c2goasm",
    sum = "h1:+n/aFZefKZp7spd8DFdX7uMikMLXX4oubIzJF4kv/wI=",
    version = "v0.0.0-20190812172519-36a3d3bbc4f3",
)

go_repository(
    name = "com_github_pierrec_lz4_v4",
    importpath = "github.com/pierrec/lz4/v4",
    sum = "h1:MO0/ucJhngq7299dKLwIMtgTfbkoSPF6AoMYDd8Q4q0=",
    version = "v4.1.15",
)

go_repository(
    name = "com_github_pkg_diff",
    importpath = "github.com/pkg/diff",
    sum = "h1:aoZm08cpOy4WuID//EZDgcC4zIxODThtZNPirFr42+A=",
    version = "v0.0.0-20210226163009-20ebb0f2a09e",
)

go_repository(
    name = "com_github_pkg_sftp",
    importpath = "github.com/pkg/sftp",
    sum = "h1:I2qBYMChEhIjOgazfJmV3/mZM256btk6wkCDRmW7JYs=",
    version = "v1.13.1",
)

go_repository(
    name = "com_github_zeebo_assert",
    importpath = "github.com/zeebo/assert",
    sum = "h1:g7C04CbJuIDKNPFHmsk4hwZDO5O+kntRxzaUoNXj+IQ=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_zeebo_xxh3",
    importpath = "github.com/zeebo/xxh3",
    sum = "h1:xZmwmqxHZA8AI603jOQ0tMqmBr9lPeFwGg6d+xy9DC0=",
    version = "v1.0.2",
)

go_repository(
    name = "com_google_cloud_go_grafeas",
    importpath = "cloud.google.com/go/grafeas",
    sum = "h1:CYjC+xzdPvbV65gi6Dr4YowKcmLo045pm18L0DhdELM=",
    version = "v0.2.0",
)

go_repository(
    name = "com_google_cloud_go_recaptchaenterprise",
    importpath = "cloud.google.com/go/recaptchaenterprise",
    sum = "h1:u6EznTGzIdsyOsvm+Xkw0aSuKFXQlyjGE9a4exk6iNQ=",
    version = "v1.3.1",
)

go_repository(
    name = "com_google_cloud_go_vision",
    importpath = "cloud.google.com/go/vision",
    sum = "h1:/CsSTkbmO9HC8iQpxbK8ATms3OQaX3YQUeTMGCxlaK4=",
    version = "v1.2.0",
)

go_repository(
    name = "com_lukechampine_uint128",
    importpath = "lukechampine.com/uint128",
    sum = "h1:mBi/5l91vocEN8otkC5bDLhi2KdCticRiwbdB0O+rjI=",
    version = "v1.2.0",
)

go_repository(
    name = "org_modernc_cc_v3",
    importpath = "modernc.org/cc/v3",
    sum = "h1:uISP3F66UlixxWEcKuIWERa4TwrZENHSL8tWxZz8bHg=",
    version = "v3.36.3",
)

go_repository(
    name = "org_modernc_ccgo_v3",
    importpath = "modernc.org/ccgo/v3",
    sum = "h1:AXquSwg7GuMk11pIdw7fmO1Y/ybgazVkMhsZWCV0mHM=",
    version = "v3.16.9",
)

go_repository(
    name = "org_modernc_ccorpus",
    importpath = "modernc.org/ccorpus",
    sum = "h1:J16RXiiqiCgua6+ZvQot4yUuUy8zxgqbqEEUuGPlISk=",
    version = "v1.11.6",
)

go_repository(
    name = "org_modernc_httpfs",
    importpath = "modernc.org/httpfs",
    sum = "h1:AAgIpFZRXuYnkjftxTAZwMIiwEqAfk8aVB2/oA6nAeM=",
    version = "v1.0.6",
)

go_repository(
    name = "org_modernc_libc",
    importpath = "modernc.org/libc",
    sum = "h1:Q8/Cpi36V/QBfuQaFVeisEBs3WqoGAJprZzmf7TfEYI=",
    version = "v1.17.1",
)

go_repository(
    name = "org_modernc_memory",
    importpath = "modernc.org/memory",
    sum = "h1:dkRh86wgmq/bJu2cAS2oqBCz/KsMZU7TUM4CibQ7eBs=",
    version = "v1.2.1",
)

go_repository(
    name = "org_modernc_opt",
    importpath = "modernc.org/opt",
    sum = "h1:3XOZf2yznlhC+ibLltsDGzABUGVx8J6pnFMS3E4dcq4=",
    version = "v0.1.3",
)

go_repository(
    name = "org_modernc_sqlite",
    importpath = "modernc.org/sqlite",
    sum = "h1:ko32eKt3jf7eqIkCgPAeHMBXw3riNSLhl2f3loEF7o8=",
    version = "v1.18.1",
)

go_repository(
    name = "org_modernc_tcl",
    importpath = "modernc.org/tcl",
    sum = "h1:npxzTwFTZYM8ghWicVIX1cRWzj7Nd8i6AqqX2p+IYao=",
    version = "v1.13.1",
)

go_repository(
    name = "org_modernc_token",
    importpath = "modernc.org/token",
    sum = "h1:a0jaWiNMDhDUtqOj09wvjWWAqd3q7WpBulmL9H2egsk=",
    version = "v1.0.0",
)

go_repository(
    name = "org_modernc_z",
    importpath = "modernc.org/z",
    sum = "h1:RTNHdsrOpeoSeOF4FbzTo8gBYByaJ5xT7NgZ9ZqRiJM=",
    version = "v1.5.1",
)

go_repository(
    name = "com_github_checkpoint_restore_go_criu_v5",
    importpath = "github.com/checkpoint-restore/go-criu/v5",
    sum = "h1:wpFFOoomK3389ue2lAb0Boag6XPht5QYpipxmSNL4d8=",
    version = "v5.3.0",
)

go_repository(
    name = "com_github_cilium_ebpf",
    importpath = "github.com/cilium/ebpf",
    sum = "h1:1k/q3ATgxSXRdrmPfH8d7YK0GfqVsEKZAX9dQZvs56k=",
    version = "v0.7.0",
)

go_repository(
    name = "com_github_moby_sys_mountinfo",
    importpath = "github.com/moby/sys/mountinfo",
    sum = "h1:2Ks8/r6lopsxWi9m58nlwjaeSzUX9iiL1vj5qB/9ObI=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_mrunalp_fileutils",
    importpath = "github.com/mrunalp/fileutils",
    sum = "h1:NKzVxiH7eSk+OQ4M+ZYW1K6h27RUV3MI6NUTsHhU6Z4=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_opencontainers_selinux",
    importpath = "github.com/opencontainers/selinux",
    sum = "h1:rAiKF8hTcgLI3w0DHm6i0ylVVcOrlgR1kK99DRLDhyU=",
    version = "v1.10.0",
)

go_repository(
    name = "com_github_seccomp_libseccomp_golang",
    importpath = "github.com/seccomp/libseccomp-golang",
    sum = "h1:RpforrEYXWkmGwJHIGnLZ3tTWStkjVVstwzNGqxX2Ds=",
    version = "v0.9.2-0.20220502022130-f33da4d89646",
)

go_repository(
    name = "com_github_vishvananda_netlink",
    importpath = "github.com/vishvananda/netlink",
    sum = "h1:1iyaYNBLmP6L0220aDnYQpo1QEV4t4hJ+xEEhhJH8j0=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_vishvananda_netns",
    importpath = "github.com/vishvananda/netns",
    sum = "h1:OviZH7qLw/7ZovXvuNyL3XQl8UFofeikI1NW1Gypu7k=",
    version = "v0.0.0-20191106174202-0a2b9b5464df",
)

go_repository(
    name = "com_github_r3labs_diff_v3",
    importpath = "github.com/r3labs/diff/v3",
    sum = "h1:CBKqf3XmNRHXKmdU7mZP1w7TV0pDyVCis1AUHtA4Xtg=",
    version = "v3.0.1",
)

go_repository(
    name = "com_github_vmihailenco_msgpack_v5",
    importpath = "github.com/vmihailenco/msgpack/v5",
    sum = "h1:5gO0H1iULLWGhs2H5tbAHIZTV8/cYafcFOr9znI5mJU=",
    version = "v5.3.5",
)

go_repository(
    name = "com_github_vmihailenco_tagparser_v2",
    importpath = "github.com/vmihailenco/tagparser/v2",
    sum = "h1:y09buUbR+b5aycVFQs/g70pqKVZNBmxwAhO7/IwNM9g=",
    version = "v2.0.0",
)

go_repository(
    name = "com_github_acomagu_bufpipe",
    importpath = "github.com/acomagu/bufpipe",
    sum = "h1:e3H4WUzM3npvo5uv95QuJM3cQspFNtFBzvJ2oNjKIDQ=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_ahmetb_gen_crd_api_reference_docs",
    importpath = "github.com/ahmetb/gen-crd-api-reference-docs",
    sum = "h1:GVJ16ekImj1gIflbV8OoBNusuBiqibDxz/G3rcFrQXw=",
    version = "v0.3.1-0.20220720053627-e327d0730470",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2",
    importpath = "github.com/aws/aws-sdk-go-v2",
    sum = "h1:shN7NlnVzvDUgPQ+1rLMSxY8OWRNDRYtiqe0p/PgrhY=",
    version = "v1.17.3",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_config",
    importpath = "github.com/aws/aws-sdk-go-v2/config",
    sum = "h1:lDpy0WM8AHsywOnVrOHaSMfpaiV2igOw8D7svkFkXVA=",
    version = "v1.18.8",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_credentials",
    importpath = "github.com/aws/aws-sdk-go-v2/credentials",
    sum = "h1:vTrwTvv5qAwjWIGhZDSBH/oQHuIQjGmD232k01FUh6A=",
    version = "v1.13.8",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_feature_ec2_imds",
    importpath = "github.com/aws/aws-sdk-go-v2/feature/ec2/imds",
    sum = "h1:j9wi1kQ8b+e0FBVHxCqCGo4kxDU175hoDHcWAi0sauU=",
    version = "v1.12.21",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_internal_configsources",
    importpath = "github.com/aws/aws-sdk-go-v2/internal/configsources",
    sum = "h1:I3cakv2Uy1vNmmhRQmFptYDxOvBnwCdNwyw63N0RaRU=",
    version = "v1.1.27",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_internal_endpoints_v2",
    importpath = "github.com/aws/aws-sdk-go-v2/internal/endpoints/v2",
    sum = "h1:5NbbMrIzmUn/TXFqAle6mgrH5m9cOvMLRGL7pnG8tRE=",
    version = "v2.4.21",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_internal_ini",
    importpath = "github.com/aws/aws-sdk-go-v2/internal/ini",
    sum = "h1:KeTxcGdNnQudb46oOl4d90f2I33DF/c6q3RnZAmvQdQ=",
    version = "v1.3.28",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_ecr",
    importpath = "github.com/aws/aws-sdk-go-v2/service/ecr",
    sum = "h1:uiF/RI+Up8H2xdgT2GWa20YzxiKEalHieqNjm6HC3Xk=",
    version = "v1.17.18",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_ecrpublic",
    importpath = "github.com/aws/aws-sdk-go-v2/service/ecrpublic",
    sum = "h1:bcQy5/dcJO8VQD+p0tDoIYdgEC3ch9f1/BNRES7XMug=",
    version = "v1.13.17",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_internal_presigned_url",
    importpath = "github.com/aws/aws-sdk-go-v2/service/internal/presigned-url",
    sum = "h1:5C6XgTViSb0bunmU57b3CT+MhxULqHH2721FVA+/kDM=",
    version = "v1.9.21",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_kms",
    importpath = "github.com/aws/aws-sdk-go-v2/service/kms",
    sum = "h1:1mEQ1BVRfxU2KzcUUIzqDQ8p6yPkhzHrHT++sjtLJts=",
    version = "v1.20.0",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_sso",
    importpath = "github.com/aws/aws-sdk-go-v2/service/sso",
    sum = "h1:/2gzjhQowRLarkkBOGPXSRnb8sQ2RVsjdG1C/UliK/c=",
    version = "v1.12.0",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_ssooidc",
    importpath = "github.com/aws/aws-sdk-go-v2/service/ssooidc",
    sum = "h1:Jfly6mRxk2ZOSlbCvZfKNS7TukSx1mIzhSsqZ/IGSZI=",
    version = "v1.14.0",
)

go_repository(
    name = "com_github_aws_aws_sdk_go_v2_service_sts",
    importpath = "github.com/aws/aws-sdk-go-v2/service/sts",
    sum = "h1:kOO++CYo50RcTFISESluhWEi5Prhg+gaSs4whWabiZU=",
    version = "v1.18.0",
)

go_repository(
    name = "com_github_aws_smithy_go",
    importpath = "github.com/aws/smithy-go",
    sum = "h1:hgz0X/DX0dGqTYpGALqXJoRKRj5oQ7150i5FdTePzO8=",
    version = "v1.13.5",
)

go_repository(
    name = "com_github_awslabs_amazon_ecr_credential_helper_ecr_login",
    importpath = "github.com/awslabs/amazon-ecr-credential-helper/ecr-login",
    sum = "h1:Ted/bR1N6ltMrASdwRhX1BrGYSFg3aeGMlK8GlgkGh4=",
    version = "v0.0.0-20221004211355-a250ad2ca1e3",
)

go_repository(
    name = "com_github_blendle_zapdriver",
    importpath = "github.com/blendle/zapdriver",
    sum = "h1:C3dydBOWYRiOk+B8X9IVZ5IOe+7cl+tGOexN4QqHfpE=",
    version = "v1.3.1",
)

go_repository(
    name = "com_github_bluekeyes_go_gitdiff",
    importpath = "github.com/bluekeyes/go-gitdiff",
    sum = "h1:w4SrRFcufU0/tEpWx3VurDBAnWfpxsmwS7yWr14meQk=",
    version = "v0.7.0",
)

go_repository(
    name = "com_github_buger_jsonparser",
    importpath = "github.com/buger/jsonparser",
    sum = "h1:2PnMjfWD7wBILjqQbt530v576A/cAbQvEW9gGIpYMUs=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_bwesterb_go_ristretto",
    importpath = "github.com/bwesterb/go-ristretto",
    sum = "h1:xxWOVbN5m8NNKiSDZXE1jtZvZnC6JSJ9cYFADiZcWtw=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_cenkalti_backoff_v3",
    importpath = "github.com/cenkalti/backoff/v3",
    sum = "h1:cfUAAO3yvKMYKPrvhDuHSwQnhZNk/RMHKdZqKTxfm6M=",
    version = "v3.2.2",
)

go_repository(
    name = "com_github_chrismellard_docker_credential_acr_env",
    importpath = "github.com/chrismellard/docker-credential-acr-env",
    sum = "h1:lG6Usi/kX/JBZzGz1H+nV+KwM97vThQeKunCbS6PutU=",
    version = "v0.0.0-20221002210726-e883f69e0206",
)

go_repository(
    name = "com_github_cjwagner_httpcache",
    importpath = "github.com/cjwagner/httpcache",
    sum = "h1:eUjwn08FDjbj8vBM31026tjBraJCu+qpDvo/q0EAvQk=",
    version = "v0.0.0-20230907212505-d4841bbad466",
)

go_repository(
    name = "com_github_cloudflare_circl",
    importpath = "github.com/cloudflare/circl",
    sum = "h1:bZgT/A+cikZnKIwn7xL2OBj012Bmvho/o6RpRvv3GKY=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_containerd_stargz_snapshotter_estargz",
    importpath = "github.com/containerd/stargz-snapshotter/estargz",
    sum = "h1:OqlDCK3ZVUO6C3B/5FSkDwbkEETK84kQgEeFwDC+62k=",
    version = "v0.14.3",
)

go_repository(
    name = "com_github_creachadair_staticfile",
    importpath = "github.com/creachadair/staticfile",
    sum = "h1:RhyrMgi7IQn3GejgmGtFuCec58vboEMt5CH6N3ulRJk=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_danwakefield_fnmatch",
    importpath = "github.com/danwakefield/fnmatch",
    sum = "h1:y5HC9v93H5EPKqaS1UYVg1uYah5Xf51mBfIoWehClUQ=",
    version = "v0.0.0-20160403171240-cbb64ac3d964",
)

go_repository(
    name = "com_github_denormal_go_gitignore",
    importpath = "github.com/denormal/go-gitignore",
    sum = "h1:0nsrg//Dc7xC74H/TZ5sYR8uk4UQRNjsw8zejqH5a4Q=",
    version = "v0.0.0-20180930084346-ae8ad1d07817",
)

go_repository(
    name = "com_github_flowstack_go_jsonschema",
    importpath = "github.com/flowstack/go-jsonschema",
    sum = "h1:dCrjGJRXIlbDsLAgTJZTjhwUJnnxVWl1OgNyYh5nyDc=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_goccy_kpoward",
    importpath = "github.com/goccy/kpoward",
    sum = "h1:UcrLMG9rq7NwrMiUc0h+qUyIlvqPzqLiPb+zQEqH8cE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_gofrs_uuid",
    importpath = "github.com/gofrs/uuid",
    sum = "h1:yyYWMnhkhrKwwr8gAOcOCYxOOscHgDS9yZgBrnJfGa0=",
    version = "v4.2.0+incompatible",
)

go_repository(
    name = "com_github_golang_jwt_jwt",
    importpath = "github.com/golang-jwt/jwt",
    sum = "h1:73Z+4BJcrTC+KczS6WvTPvRGOp1WmfEP4Q1lOd9Z/+c=",
    version = "v3.2.1+incompatible",
)

go_repository(
    name = "com_github_golang_jwt_jwt_v4",
    importpath = "github.com/golang-jwt/jwt/v4",
    sum = "h1:7cYmW1XlMY7h7ii7UhUyChSgS5wUJEnm9uZVTGqOWzg=",
    version = "v4.5.0",
)

go_repository(
    name = "com_github_google_go_containerregistry_pkg_authn_k8schain",
    importpath = "github.com/google/go-containerregistry/pkg/authn/k8schain",
    sum = "h1:GtokKAIyg0S9Y/60CO+gHUBHwYblbwfaUS3Qr+AejTc=",
    version = "v0.0.0-20221030203717-1711cefd7eec",
)

go_repository(
    name = "com_github_google_go_containerregistry_pkg_authn_kubernetes",
    importpath = "github.com/google/go-containerregistry/pkg/authn/kubernetes",
    sum = "h1:+nq85YWt99EkBpsKV+ABoAzxM7My/uOKHModpV/mwgs=",
    version = "v0.0.0-20221017135236-9b4fdd506cdd",
)

go_repository(
    name = "com_github_google_s2a_go",
    importpath = "github.com/google/s2a-go",
    sum = "h1:FAgZmpLl/SXurPEZyCMPBIiiYeTbqfjlbdnCNTAkbGE=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_hashicorp_go_secure_stdlib_mlock",
    importpath = "github.com/hashicorp/go-secure-stdlib/mlock",
    sum = "h1:p4AKXPPS24tO8Wc8i1gLvSKdmkiSY5xuju57czJ/IJQ=",
    version = "v0.1.2",
)

go_repository(
    name = "com_github_hashicorp_go_secure_stdlib_parseutil",
    importpath = "github.com/hashicorp/go-secure-stdlib/parseutil",
    sum = "h1:UpiO20jno/eV1eVZcxqWnUohyKRe1g8FPV/xH1s/2qs=",
    version = "v0.1.7",
)

go_repository(
    name = "com_github_hashicorp_go_secure_stdlib_strutil",
    importpath = "github.com/hashicorp/go-secure-stdlib/strutil",
    sum = "h1:kes8mmyCpxJsI7FTwtzRqEy9CdjCtrXrXGuOpxEA7Ts=",
    version = "v0.1.2",
)

go_repository(
    name = "com_github_jellydator_ttlcache_v2",
    importpath = "github.com/jellydator/ttlcache/v2",
    sum = "h1:AZGME43Eh2Vv3giG6GeqeLeFXxwxn1/qHItqWZl6U64=",
    version = "v2.11.1",
)

go_repository(
    name = "com_github_letsencrypt_boulder",
    importpath = "github.com/letsencrypt/boulder",
    sum = "h1:ndns1qx/5dL43g16EQkPV/i8+b3l5bYQwLeoSBe7tS8=",
    version = "v0.0.0-20221109233200-85aa52084eaf",
)

go_repository(
    name = "com_github_matryer_is",
    importpath = "github.com/matryer/is",
    sum = "h1:92UTHpy8CDwaJ08GqLDzhhuixiBUUD1p3AU6PHddz4A=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_mmcloughlin_avo",
    importpath = "github.com/mmcloughlin/avo",
    sum = "h1:nAco9/aI9Lg2kiuROBY6BhCI/z0t5jEvJfjWbL8qXLU=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_pjbgf_sha1cd",
    importpath = "github.com/pjbgf/sha1cd",
    sum = "h1:4D5XXmUUBUl/xQ6IjCkEAbqXskkq/4O7LmGn0AqMDs4=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_prometheus_statsd_exporter",
    importpath = "github.com/prometheus/statsd_exporter",
    sum = "h1:hA05Q5RFeIjgwKIYEdFd59xu5Wwaznf33yKI+pyX6T8=",
    version = "v0.21.0",
)

go_repository(
    name = "com_github_protonmail_go_crypto",
    importpath = "github.com/ProtonMail/go-crypto",
    sum = "h1:wPbRQzjjwFc0ih8puEVAOFGELsn1zoIIYdxvML7mDxA=",
    version = "v0.0.0-20230217124315-7d5c6f04bbb8",
)

go_repository(
    name = "com_github_sigstore_sigstore",
    importpath = "github.com/sigstore/sigstore",
    sum = "h1:iUou0QJW8eQKMUkTXbFyof9ZOblDtfaW2Sn2+QI8Tcs=",
    version = "v1.5.1",
)

go_repository(
    name = "com_github_skeema_knownhosts",
    importpath = "github.com/skeema/knownhosts",
    sum = "h1:Wvr9V0MxhjRbl3f9nMnKnFfiWTJmtECJ9Njkea3ysW0=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_smarty_assertions",
    importpath = "github.com/smarty/assertions",
    sum = "h1:812oFiXI+G55vxsFf+8bIZ1ux30qtkdqzKbEFwyX3Tk=",
    version = "v1.15.1",
)

go_repository(
    name = "com_github_spiffe_go_spiffe_v2",
    importpath = "github.com/spiffe/go-spiffe/v2",
    sum = "h1:nfNwopOP7q0qsWU6AUASqmbtYViwHA6vuHyAtqFJtNc=",
    version = "v2.1.2",
)

go_repository(
    name = "com_github_spiffe_spire_api_sdk",
    importpath = "github.com/spiffe/spire-api-sdk",
    sum = "h1:j1EB0YjZXeq69u3Hrzw1FkMqPepBky9qvuAXdV9Gbyo=",
    version = "v1.5.4",
)

go_repository(
    name = "com_github_theupdateframework_go_tuf",
    importpath = "github.com/theupdateframework/go-tuf",
    sum = "h1:1i/Afw3rmaR1gF3sfVkG2X6ldkikQwA9zY380LrR5YI=",
    version = "v0.5.2-0.20220930112810-3890c1e7ace4",
)

go_repository(
    name = "com_github_titanous_rocacheck",
    importpath = "github.com/titanous/rocacheck",
    sum = "h1:e/5i7d4oYZ+C1wj2THlRK+oAhjeS/TRQwMfkIuet3w0=",
    version = "v0.0.0-20171023193734-afe73141d399",
)

go_repository(
    name = "com_github_tsenart_vegeta_v12",
    importpath = "github.com/tsenart/vegeta/v12",
    sum = "h1:UQ7tG7WkDorKj0wjx78Z4/vsMBP8RJQMGJqRVrkvngg=",
    version = "v12.8.4",
)

go_repository(
    name = "com_github_vbatts_tar_split",
    importpath = "github.com/vbatts/tar-split",
    sum = "h1:hLFqsOLQ1SsppQNTMpkpPXClLDfC2A3Zgy9OUU+RVck=",
    version = "v0.11.3",
)

go_repository(
    name = "com_github_zeebo_errs",
    importpath = "github.com/zeebo/errs",
    sum = "h1:hmiaKqgYZzcVgRL1Vkc1Mn2914BbzB0IBxs+ebeutGs=",
    version = "v1.3.0",
)

go_repository(
    name = "dev_knative_hack",
    importpath = "knative.dev/hack",
    sum = "h1:CDa7s9KspEZqPhk7cN68ZypRLuAvSgr+knoOaXSsrHk=",
    version = "v0.0.0-20230113013652-c7cfcb062de9",
)

go_repository(
    name = "io_opentelemetry_go_otel_exporters_jaeger",
    importpath = "go.opentelemetry.io/otel/exporters/jaeger",
    sum = "h1:VAMoGujbVV8Q0JNM/cEbhzUIWWBxnEqH45HP9iBKN04=",
    version = "v1.13.0",
)

go_repository(
    name = "org_bitbucket_creachadair_stringset",
    importpath = "bitbucket.org/creachadair/stringset",
    sum = "h1:L4vld9nzPt90UZNrXjNelTshD74ps4P5NGs3Iq6yN3o=",
    version = "v0.0.9",
)

go_repository(
    name = "org_golang_x_arch",
    importpath = "golang.org/x/arch",
    sum = "h1:oMxhUYsO9VsR1dcoVUjJjIGhx1LXol3989T/yZ59Xsw=",
    version = "v0.1.0",
)

go_repository(
    name = "org_uber_go_automaxprocs",
    importpath = "go.uber.org/automaxprocs",
    sum = "h1:CpDZl6aOlLhReez+8S3eEotD7Jx0Os++lemPlMULQP0=",
    version = "v1.4.0",
)

gazelle_dependencies()

register_toolchains("//:py_toolchain")
