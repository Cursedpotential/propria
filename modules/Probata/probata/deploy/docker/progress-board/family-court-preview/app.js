const viewButtons = Array.from(document.querySelectorAll("[data-view]"));
const panels = Array.from(document.querySelectorAll("[data-panel]"));
const themeToggle = document.querySelector("#theme-toggle");

function selectView(view) {
  const known = panels.some((panel) => panel.dataset.panel === view);
  const selected = known ? view : "dashboard";

  for (const button of viewButtons) {
    const active = button.dataset.view === selected;
    button.classList.toggle("is-active", active);
    if (active) {
      button.setAttribute("aria-current", "page");
    } else {
      button.removeAttribute("aria-current");
    }
  }

  for (const panel of panels) {
    const active = panel.dataset.panel === selected;
    panel.hidden = !active;
    panel.classList.toggle("is-visible", active);
  }

  if (window.location.hash !== `#${selected}`) {
    history.replaceState(null, "", `#${selected}`);
  }
}

for (const button of viewButtons) {
  button.addEventListener("click", () => selectView(button.dataset.view));
}

themeToggle.addEventListener("click", () => {
  const dark = document.documentElement.dataset.theme !== "dark";
  document.documentElement.dataset.theme = dark ? "dark" : "light";
  themeToggle.setAttribute("aria-pressed", String(dark));
  themeToggle.textContent = dark ? "Light preview" : "Dark preview";
});

window.addEventListener("hashchange", () => selectView(window.location.hash.slice(1)));
selectView(window.location.hash.slice(1));
