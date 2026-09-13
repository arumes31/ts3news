document.addEventListener('DOMContentLoaded', function () {
  document.querySelectorAll('[data-password-toggle]').forEach(function (button) {
    button.addEventListener('click', function () {
      var input = document.getElementById(button.dataset.passwordToggle);
      var visible = input.type === 'password';
      input.type = visible ? 'text' : 'password';
      button.textContent = visible ? 'Hide' : 'Show';
      button.setAttribute('aria-pressed', String(visible));
    });
  });
  var copy = document.querySelector('[data-copy-username]');
  if (copy) copy.addEventListener('click', async function () {
    var input = document.getElementById('accountUsername');
    var status = document.getElementById('accountCopyStatus');
    try {
      await navigator.clipboard.writeText(input.value);
      status.textContent = 'Username copied.';
    } catch (_) {
      input.focus(); input.select();
      status.textContent = 'Username selected. Copy it using your device’s copy action.';
    }
  });
});
