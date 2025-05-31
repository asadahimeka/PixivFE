# usage: nix-shell

let
  nixpkgs = fetchTarball "https://nixos.org/channels/nixos-unstable/nixexprs.tar.xz";
  pkgs = import nixpkgs { config = {}; overlays = []; };
in

pkgs.mkShellNoCC {
  packages = with pkgs; [
    tailwindcss_4
    go
    jq
    crowdin-cli

    tmux
    watchman # needed for tailwind
  ];

  shellHook = ''
   tailwind() {
           tmux new-session -d -s tailwind
           tmux send-keys -t tailwind "tailwindcss -i assets/css/tailwind-style_source.css -o assets/css/tailwind-style.css --watch --minify" ENTER
           trap "tmux kill-session -t tailwind" EXIT
   }

   tailwind

   echo "Tailwind CSS daemon is running in tmux (attach with 'tmux a')"
   echo "The Tailwind daemon will be terminated on shell exit."
   echo "You may start running PixivFE now. Example: ./build.sh run"
  '';
}
