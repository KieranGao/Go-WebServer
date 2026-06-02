/**
 * custom.js - Custom JavaScript Utilities
 * Provides common utilities and helper functions
 */

(function() {
    'use strict';

    // Custom namespace
    window.CustomUtils = window.CustomUtils || {};

    /**
     * Utility: Log messages with timestamp
     */
    CustomUtils.log = function(message, type) {
        type = type || 'log';
        var timestamp = new Date().toLocaleTimeString();
        console[type]('[' + timestamp + '] ' + message);
    };

    /**
     * Utility: Smooth scroll to element
     */
    CustomUtils.smoothScroll = function(element) {
        if (typeof element === 'string') {
            element = document.querySelector(element);
        }
        if (element) {
            element.scrollIntoView({ behavior: 'smooth' });
        }
    };

    /**
     * Utility: Debounce function
     */
    CustomUtils.debounce = function(func, wait) {
        var timeout;
        return function executedFunction() {
            var later = function() {
                clearTimeout(timeout);
                func();
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    };

    /**
     * Utility: Add active class to navigation
     */
    CustomUtils.setActiveNavigation = function(current) {
        var navLinks = document.querySelectorAll('nav a');
        navLinks.forEach(function(link) {
            link.classList.remove('active');
            if (link.getAttribute('href') === current) {
                link.classList.add('active');
            }
        });
    };

    /**
     * Utility: Parse query string
     */
    CustomUtils.parseQueryString = function() {
        var params = {};
        var queryString = window.location.search.substring(1);
        var queries = queryString.split('&');
        queries.forEach(function(query) {
            var pair = query.split('=');
            params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1] || '');
        });
        return params;
    };

    // Log initialization
    CustomUtils.log('CustomUtils loaded');

})();
