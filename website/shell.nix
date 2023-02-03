{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  # nativeBuildInputs is usually what you want -- tools you need to run
  nativeBuildInputs = with pkgs; [ go_1_19 hugo postcss-cli nodejs ];
}
