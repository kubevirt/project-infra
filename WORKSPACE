load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "rules_python",
    sha256 = "934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
    url = "https://github.com/bazelbuild/rules_python/releases/download/0.3.0/rules_python-0.3.0.tar.gz",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "59d5b42ac315e7eadffa944e86e90c2990110a1c8075f1cd145f487e999d22b3",
    strip_prefix = "rules_docker-0.17.0",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.17.0/rules_docker-v0.17.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "099a9fb96a376ccbbb7d291ed4ecbdfd42f6bc822ab77ae6f1b5cb9e914e94fa",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.35.0/rules_go-v0.35.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.35.0/rules_go-v0.35.0.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "efbbba6ac1a4fd342d5122cbdfdb82aeb2cf2862e35022c752eaddffada7c3f3",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.27.0/bazel-gazelle-v0.27.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.27.0/bazel-gazelle-v0.27.0.tar.gz",
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

go_register_toolchains(
    go_version = "1.19.1",
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
    name = "com_github_bndr_gotabulate",
    importpath = "github.com/bndr/gotabulate",
    sum = "h1:yC9izuZEphojb9r+KYL4W9IJKO/ceIO8HDwxMA24U4c=",
    version = "v1.1.2",
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
    sum = "h1:EroSdlP9BOoL5ssLYf3uLJXhCQMMM2fFxCJDKA3RhnA=",
    version = "v1.0.0",
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

gazelle_dependencies()

register_toolchains("//:py_toolchain")
