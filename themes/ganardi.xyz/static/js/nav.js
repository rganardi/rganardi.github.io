(() => {
'use strict';

  document
    .getElementById('nav-toggle')
    .addEventListener('click', () => {
      document.getElementById('nav-links').classList.toggle('hide');
      document.getElementById('nav-toggle').classList.toggle('active');
      if (document.getElementById('nav-back')) {
        document.getElementById('nav-back').classList.toggle('active');
      }
    });
})();
