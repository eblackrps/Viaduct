group "default" {
  targets = ["image"]
}

target "docker-metadata-action" {}

target "image" {
  inherits = ["docker-metadata-action"]
  context = "."
  dockerfile = "Dockerfile"
  pull = true
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
}
