function OpenPreviewer(url) {
  const viewer = document.createElement("div");
  viewer.style.cssText = `
    height: 100vh;
    width: 100vw;
    position: fixed;
    top: 0;
    left: 0;
    background: rgba(0,0,0,.8);
    display: flex;
    flex-direction: column;
    padding: 0 3rem;
    overflow: scroll;
    z-index: 999;
  `;

  const imageLink = url
    .replace(/c\/\d+x\d+.*?\//, "")
    .replace(/square1200/, "master1200");
  const img = document.createElement("img");
  img.src = imageLink;
  img.style.cssText = `
    margin: 3rem auto;
    max-width: 90%;
    max-height: 90%;
  `;

  viewer.appendChild(img);
  document.body.appendChild(viewer);

  viewer.onclick = () => {
    document.body.removeChild(viewer);
  };
}

function AddOverlay() {
  // This is a workaround for cookies having the `HttpOnly` attribute set
  const type = document.querySelector("#artworkPreview").innerHTML;

  let className, html;

  if (type === "cover") {
    className = "absolute size-full z-5 inset-0";
    html = "";
  } else if (type === "button") {
    className =
      "absolute size-fit p-1 z-5 bottom-0 right-0 rounded bg-neutral-950";
    html = "↗";
  } else {
    return;
  }

  document.querySelectorAll(".media-target img").forEach((illust) => {
    const url = illust.getAttribute("src");
    // Skip processing NSFW placeholder images
    if (url.includes("/img/nsfw.png")) return;

    const button = document.createElement("div");
    button.setAttribute("class", className);
    button.innerHTML = html;

    const mediaTarget = illust.closest(".media-target");
    if (mediaTarget) {
      mediaTarget.appendChild(button);

      button.onclick = (e) => {
        e.preventDefault();
        e.stopPropagation();
        OpenPreviewer(url);
      };
    }
  });
}

addEventListener("htmx:afterSwap", function (event) {
  // console.log("%o", event);
  AddOverlay();
});

// Initialize (it will only run one time)
AddOverlay();
